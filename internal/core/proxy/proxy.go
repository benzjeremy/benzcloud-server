package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/benzjeremy/benzcloud-server/internal/core/plugins"
)

// Router handles incoming HTTP requests and dispatches them based on Subdomain/Host.
type Router struct {
	baseDomain    string
	serverLocalIP string
	serverToken   string
	pluginMgr     *plugins.Manager
	mainHandler   http.Handler
	driveHandler  http.Handler
	proxies       map[int]*httputil.ReverseProxy
}

// NewRouter creates the virtual host dispatcher.
func NewRouter(baseDomain, serverLocalIP, serverToken string, pluginMgr *plugins.Manager, mainHandler, driveHandler http.Handler) *Router {
	return &Router{
		baseDomain:    strings.ToLower(strings.Trim(baseDomain, ".")),
		serverLocalIP: serverLocalIP,
		serverToken:   serverToken,
		pluginMgr:     pluginMgr,
		mainHandler:   mainHandler,
		driveHandler:  driveHandler,
		proxies:       make(map[int]*httputil.ReverseProxy),
	}
}

func (rt *Router) getReverseProxy(port int) *httputil.ReverseProxy {
	if p, exists := rt.proxies[port]; exists {
		return p
	}
	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	// Add server token header for internal authentication
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Header.Set("X-BenzCloud-Token", rt.serverToken)
	}
	rt.proxies[port] = proxy
	return proxy
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Security Headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	host := strings.ToLower(r.Host)
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// 1. Direct IP or localhost or root baseDomain -> Main Handler (Setup / Dashboard / API)
	if host == "localhost" || host == "127.0.0.1" || host == rt.serverLocalIP || host == "10.42.0.1" || host == rt.baseDomain {
		rt.mainHandler.ServeHTTP(w, r)
		return
	}

	// 2. Subdomain check: host must end with .baseDomain
	suffix := "." + rt.baseDomain
	if !strings.HasSuffix(host, suffix) {
		// Anti-DNS Rebinding: Unknown external Host header
		http.Error(w, "403 Forbidden: Invalid Host Header (Anti-DNS-Rebinding Protection)", http.StatusForbidden)
		return
	}

	subdomain := strings.TrimSuffix(host, suffix)

	// 3. System Subdomains
	switch subdomain {
	case "drive":
		if rt.driveHandler != nil {
			rt.driveHandler.ServeHTTP(w, r)
			return
		}
	case "vpn":
		// Direct to VPN status on main handler
		r.URL.Path = "/#vpn"
		rt.mainHandler.ServeHTTP(w, r)
		return
	}

	// 4. Plugin Subdomains (mail, chat, or custom web)
	if rt.pluginMgr != nil {
		if port, ok := rt.pluginMgr.GetTargetForSubdomain(subdomain); ok {
			proxy := rt.getReverseProxy(port)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	// Subdomain not found
	http.Error(w, fmt.Sprintf("404 BenzCloud: Subdomain '%s.%s' is not mapped to any active service or plugin.", subdomain, rt.baseDomain), http.StatusNotFound)
}
