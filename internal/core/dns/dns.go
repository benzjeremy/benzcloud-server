package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// QueryLogEntry records a DNS query for the live dashboard.
type QueryLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ClientIP  string    `json:"client_ip"`
	Domain    string    `json:"domain"`
	Type      string    `json:"type"`
	Answer    string    `json:"answer"`
	Forwarded bool      `json:"forwarded"`
}

// Server is the built-in lightweight DNS server.
type Server struct {
	baseDomain  string
	targetIP    string
	listenPort  int
	records     map[string]string // FQDN (lowercase, ending in dot) -> IP
	upstreams   []string
	queryLog    []QueryLogEntry
	maxLogSize  int
	udpServer   *dns.Server
	tcpServer   *dns.Server
	running     bool
	mu          sync.RWMutex
}

// NewServer creates a new DNS server instance.
func NewServer(baseDomain, targetIP string, listenPort int) *Server {
	cleanBase := strings.Trim(baseDomain, ".")
	s := &Server{
		baseDomain: cleanBase,
		targetIP:   targetIP,
		listenPort: listenPort,
		records:    make(map[string]string),
		upstreams:  []string{"9.9.9.9:53", "1.1.1.1:53"},
		queryLog:   make([]QueryLogEntry, 0, 500),
		maxLogSize: 500,
	}

	// Pre-seed system subdomains
	s.registerDefaultRecords()
	return s
}

func (s *Server) registerDefaultRecords() {
	domainDot := strings.ToLower(s.baseDomain) + "."
	// Root domain
	s.records[domainDot] = s.targetIP
	// System subdomains
	s.records["vpn."+domainDot] = s.targetIP
	s.records["drive."+domainDot] = s.targetIP
	s.records["mail."+domainDot] = s.targetIP
	s.records["chat."+domainDot] = s.targetIP
}

// RegisterSubdomain dynamically registers a custom subdomain.
// E.g. sub="blog", ip="10.42.0.1" -> blog.<domain>.
func (s *Server) RegisterSubdomain(sub, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fqdn := strings.ToLower(sub) + "." + strings.ToLower(s.baseDomain) + "."
	s.records[fqdn] = ip
}

// UnregisterSubdomain removes a custom subdomain.
func (s *Server) UnregisterSubdomain(sub string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fqdn := strings.ToLower(sub) + "." + strings.ToLower(s.baseDomain) + "."
	delete(s.records, fqdn)
}

// GetRecords returns a snapshot of all registered DNS records.
func (s *Server) GetRecords() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]string, len(s.records))
	for k, v := range s.records {
		res[k] = v
	}
	return res
}

// GetQueryLog returns the recent DNS queries.
func (s *Server) GetQueryLog() []QueryLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]QueryLogEntry, len(s.queryLog))
	copy(entries, s.queryLog)
	return entries
}

func (s *Server) logQuery(entry QueryLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queryLog) >= s.maxLogSize {
		s.queryLog = s.queryLog[1:]
	}
	s.queryLog = append(s.queryLog, entry)
}

// ServeDNS handles incoming DNS requests.
func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	clientIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())

	for _, q := range r.Question {
		qName := strings.ToLower(q.Name)
		qType := dns.TypeToString[q.Qtype]

		s.mu.RLock()
		ip, isLocal := s.records[qName]
		baseMatch := strings.HasSuffix(qName, strings.ToLower(s.baseDomain)+".")
		s.mu.RUnlock()

		if isLocal {
			if q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY {
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil {
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    60,
						},
						A: parsedIP.To4(),
					}
					msg.Answer = append(msg.Answer, rr)
				}
			}
			s.logQuery(QueryLogEntry{
				Timestamp: time.Now().UTC(),
				ClientIP:  clientIP,
				Domain:    q.Name,
				Type:      qType,
				Answer:    ip,
				Forwarded: false,
			})
		} else if baseMatch {
			// Subdomain not found under base domain -> NXDOMAIN
			msg.Rcode = dns.RcodeNameError
			s.logQuery(QueryLogEntry{
				Timestamp: time.Now().UTC(),
				ClientIP:  clientIP,
				Domain:    q.Name,
				Type:      qType,
				Answer:    "NXDOMAIN",
				Forwarded: false,
			})
		} else {
			// External query -> forward to upstream resolver
			upstreamResp, err := s.forwardQuery(r)
			if err == nil && upstreamResp != nil {
				_ = w.WriteMsg(upstreamResp)
				var ansStr string
				if len(upstreamResp.Answer) > 0 {
					ansStr = upstreamResp.Answer[0].String()
				}
				s.logQuery(QueryLogEntry{
					Timestamp: time.Now().UTC(),
					ClientIP:  clientIP,
					Domain:    q.Name,
					Type:      qType,
					Answer:    ansStr,
					Forwarded: true,
				})
				return
			}
			msg.Rcode = dns.RcodeServerFailure
		}
	}

	_ = w.WriteMsg(msg)
}

func (s *Server) forwardQuery(r *dns.Msg) (*dns.Msg, error) {
	c := new(dns.Client)
	c.Timeout = 2 * time.Second

	for _, upstream := range s.upstreams {
		resp, _, err := c.Exchange(r, upstream)
		if err == nil && resp != nil {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("all upstream resolvers failed")
}

// Start launches the UDP and TCP listeners.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	listenAddr := fmt.Sprintf("0.0.0.0:%d", s.listenPort)
	s.mu.Unlock()

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", s.ServeDNS)
	s.udpServer = &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: udpHandler,
	}

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", s.ServeDNS)
	s.tcpServer = &dns.Server{
		Addr:    listenAddr,
		Net:     "tcp",
		Handler: tcpHandler,
	}

	errChan := make(chan error, 2)

	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("DNS UDP failed: %w", err)
		}
	}()

	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("DNS TCP failed: %w", err)
		}
	}()

	// Brief wait to detect immediate bind failures (e.g. port already in use or permission denied)
	select {
	case err := <-errChan:
		_ = s.Stop()
		return err
	case <-time.After(150 * time.Millisecond):
		log.Printf("[BenzCloud DNS] Server actively listening on %s (UDP & TCP)", listenAddr)
		return nil
	}
}

// Stop gracefully shuts down DNS listeners.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false

	var errs []string
	if s.udpServer != nil {
		if err := s.udpServer.ShutdownContext(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if s.tcpServer != nil {
		if err := s.tcpServer.ShutdownContext(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping DNS: %s", strings.Join(errs, ", "))
	}
	return nil
}
