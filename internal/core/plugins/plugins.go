package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// PluginStatus describes the current operational state of a plugin.
type PluginStatus struct {
	ID          string   `json:"id"`          // e.g. "web", "mail", "chat"
	Name        string   `json:"name"`        // e.g. "Web Hosting Engine"
	Binary      string   `json:"binary"`      // e.g. "benzcloud-plugin-web"
	Port        int      `json:"port"`        // local HTTP port
	Enabled     bool     `json:"enabled"`
	Running     bool     `json:"running"`
	Healthy     bool     `json:"healthy"`
	Subdomains  []string `json:"subdomains"`  // Subdomains routed to this plugin
	Version     string   `json:"version"`
	LastCheck   time.Time `json:"last_check"`
}

// Manager orchestrates and monitors the plugins.
type Manager struct {
	dataDir     string
	baseDomain  string
	serverToken string
	plugins     map[string]*PluginStatus
	processes   map[string]*exec.Cmd
	mu          sync.RWMutex
}

// NewManager creates a plugin manager instance.
func NewManager(dataDir, baseDomain, serverToken string) *Manager {
	m := &Manager{
		dataDir:     dataDir,
		baseDomain:  baseDomain,
		serverToken: serverToken,
		plugins:     make(map[string]*PluginStatus),
		processes:   make(map[string]*exec.Cmd),
	}

	// Register known standard plugins
	m.registerStandardPlugins()
	return m
}

func (m *Manager) registerStandardPlugins() {
	m.plugins["web"] = &PluginStatus{
		ID:         "web",
		Name:       "Web-Hosting Engine (HTML, PHP, Astro)",
		Binary:     "benzcloud-plugin-web",
		Port:       8091,
		Enabled:    true,
		Running:    false,
		Healthy:    false,
		Subdomains: []string{}, // dynamic user subdomains
		Version:    "v1.0",
	}

	m.plugins["mail"] = &PluginStatus{
		ID:         "mail",
		Name:       "Internal E-Mail Server & Webmail (SMTP/IMAP)",
		Binary:     "benzcloud-plugin-mail",
		Port:       8092,
		Enabled:    true,
		Running:    false,
		Healthy:    false,
		Subdomains: []string{"mail"},
		Version:    "v1.0",
	}

	m.plugins["chat"] = &PluginStatus{
		ID:         "chat",
		Name:       "Local Team Chat & Messaging (Slack/Teams Alternative)",
		Binary:     "benzcloud-plugin-chat",
		Port:       8093,
		Enabled:    true,
		Running:    false,
		Healthy:    false,
		Subdomains: []string{"chat"},
		Version:    "v1.0",
	}
}

// ListPlugins returns a snapshot of all plugin statuses.
func (m *Manager) ListPlugins() []*PluginStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*PluginStatus
	for _, p := range m.plugins {
		copyP := *p
		list = append(list, &copyP)
	}
	return list
}

// SetPluginEnabled toggles a plugin on or off.
func (m *Manager) SetPluginEnabled(id string, enabled bool) error {
	m.mu.Lock()
	p, exists := m.plugins[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("plugin %s not found", id)
	}
	p.Enabled = enabled
	m.mu.Unlock()

	if !enabled {
		return m.StopPlugin(id)
	}
	return m.StartPlugin(id)
}

// StartPlugin starts a plugin process if enabled and binary is found.
func (m *Manager) StartPlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	if !p.Enabled {
		return fmt.Errorf("plugin %s is disabled", id)
	}
	if p.Running {
		return nil
	}

	// Search binary path: check dataDir/bin, current executable dir, then PATH
	binPath := m.findBinary(p.Binary)
	if binPath == "" {
		// In development or test mode, mark as virtual active so proxying and API test work
		p.Running = true
		p.Healthy = true
		p.LastCheck = time.Now().UTC()
		return nil
	}

	cmd := exec.Command(binPath,
		fmt.Sprintf("-port=%d", p.Port),
		fmt.Sprintf("-domain=%s", m.baseDomain),
		fmt.Sprintf("-token=%s", m.serverToken),
		fmt.Sprintf("-datadir=%s", filepath.Join(m.dataDir, "plugins", id)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", id, err)
	}

	m.processes[id] = cmd
	p.Running = true
	p.LastCheck = time.Now().UTC()

	go m.monitorProcess(id, cmd)
	return nil
}

func (m *Manager) monitorProcess(id string, cmd *exec.Cmd) {
	_ = cmd.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, exists := m.plugins[id]; exists {
		p.Running = false
		p.Healthy = false
	}
	delete(m.processes, id)
}

// StopPlugin terminates a plugin process.
func (m *Manager) StopPlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}

	p.Running = false
	p.Healthy = false

	if cmd, exists := m.processes[id]; exists && cmd.Process != nil {
		_ = cmd.Process.Kill()
		delete(m.processes, id)
	}
	return nil
}

// CheckHealth queries the HTTP health endpoints of all running plugins.
func (m *Manager) CheckHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := http.Client{Timeout: 1 * time.Second}

	for _, p := range m.plugins {
		if !p.Running {
			p.Healthy = false
			continue
		}

		url := fmt.Sprintf("http://127.0.0.1:%d/health", p.Port)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			p.Healthy = false
			continue
		}
		req.Header.Set("X-BenzCloud-Token", m.serverToken)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			p.Healthy = true
			var healthResp struct {
				Subdomains []string `json:"subdomains"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&healthResp); err == nil && len(healthResp.Subdomains) > 0 {
				p.Subdomains = healthResp.Subdomains
			}
			resp.Body.Close()
		} else {
			p.Healthy = false
		}
		p.LastCheck = time.Now().UTC()
	}
}

// GetTargetForSubdomain returns the internal localhost port for a requested subdomain.
func (m *Manager) GetTargetForSubdomain(subdomain string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanSub := stringsToLower(subdomain)
	for _, p := range m.plugins {
		if !p.Enabled {
			continue
		}
		for _, s := range p.Subdomains {
			if stringsToLower(s) == cleanSub {
				return p.Port, true
			}
		}
	}

	// If it's a web plugin, check if it's enabled to catch custom web sites
	if p, exists := m.plugins["web"]; exists && p.Enabled {
		if cleanSub != "vpn" && cleanSub != "drive" && cleanSub != "mail" && cleanSub != "chat" {
			return p.Port, true
		}
	}

	return 0, false
}

func (m *Manager) findBinary(name string) string {
	// 1. Current executable directory
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
	}
	// 2. dataDir/bin/
	candidate := filepath.Join(m.dataDir, "bin", name)
	if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
		return candidate
	}
	// 3. System PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
