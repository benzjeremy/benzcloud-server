package dns

import (
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestDNSServerResolution(t *testing.T) {
	testPort := 15353
	baseDomain := "benzjeremy.de"
	targetIP := "10.42.0.1"

	srv := NewServer(baseDomain, targetIP, testPort)
	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start DNS server on port %d: %v", testPort, err)
	}
	defer srv.Stop()

	// Give server time to bind
	time.Sleep(100 * time.Millisecond)

	c := new(dns.Client)

	// Test 1: Query drive.benzjeremy.de
	m := new(dns.Msg)
	m.SetQuestion("drive.benzjeremy.de.", dns.TypeA)
	in, _, err := c.Exchange(m, "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}
	if len(in.Answer) == 0 {
		t.Fatal("Expected at least 1 answer for drive.benzjeremy.de, got 0")
	}
	if aRec, ok := in.Answer[0].(*dns.A); !ok || aRec.A.String() != targetIP {
		t.Fatalf("Expected IP %s, got %v", targetIP, in.Answer[0])
	}

	// Test 2: Register custom subdomain and query it
	srv.RegisterSubdomain("mein-shop", "10.42.0.1")
	m2 := new(dns.Msg)
	m2.SetQuestion("mein-shop.benzjeremy.de.", dns.TypeA)
	in2, _, err := c.Exchange(m2, "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("Custom subdomain query failed: %v", err)
	}
	if len(in2.Answer) == 0 {
		t.Fatal("Expected answer for mein-shop.benzjeremy.de")
	}
	if aRec, ok := in2.Answer[0].(*dns.A); !ok || aRec.A.String() != targetIP {
		t.Fatalf("Expected IP %s, got %v", targetIP, in2.Answer[0])
	}

	// Test 3: Query unknown subdomain under base domain -> NXDOMAIN
	m3 := new(dns.Msg)
	m3.SetQuestion("gibtsnicht.benzjeremy.de.", dns.TypeA)
	in3, _, err := c.Exchange(m3, "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("NXDOMAIN query failed: %v", err)
	}
	if in3.Rcode != dns.RcodeNameError {
		t.Fatalf("Expected RcodeNameError, got %d", in3.Rcode)
	}

	// Test 4: Query log contains entries
	logs := srv.GetQueryLog()
	if len(logs) < 3 {
		t.Fatalf("Expected at least 3 query log entries, got %d", len(logs))
	}
	if !strings.Contains(logs[0].Domain, "drive.benzjeremy.de") {
		t.Fatalf("Unexpected first query in log: %s", logs[0].Domain)
	}
}
