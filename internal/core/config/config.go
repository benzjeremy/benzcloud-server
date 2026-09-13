package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// Config holds the complete state and configuration for benzcloud-server.
type Config struct {
	BaseDomain     string `json:"base_domain"`     // e.g. benzjeremy.de or intern
	ServerLocalIP  string `json:"server_local_ip"` // LAN IP, e.g. 192.168.0.5
	HTTPPort       int    `json:"http_port"`       // default 80
	DNSPort        int    `json:"dns_port"`        // default 53 (or 5353 fallback)
	VPNPort        int    `json:"vpn_port"`        // default 4242 (UDP)
	OverlaySubnet  string `json:"overlay_subnet"`  // e.g. 10.42.0.0/16
	ServerVPNIP    string `json:"server_vpn_ip"`   // e.g. 10.42.0.1
	SetupCompleted bool   `json:"setup_completed"`
	DataDir        string `json:"data_dir"`
	MasterSalt     string `json:"master_salt"` // hex encoded salt for PBKDF2
	ServerToken    string `json:"server_token"` // 32-byte CSPRNG token
	mu             sync.RWMutex
}

var ErrNotConfigured = errors.New("benzcloud-server is not configured yet (setup pending)")

// DefaultConfig returns reasonable default settings.
func DefaultConfig(dataDir string) *Config {
	localIP := detectLocalIP()
	return &Config{
		BaseDomain:     "intern",
		ServerLocalIP:  localIP,
		HTTPPort:       80,
		DNSPort:        53,
		VPNPort:        4242,
		OverlaySubnet:  "10.42.0.0/16",
		ServerVPNIP:    "10.42.0.1",
		SetupCompleted: false,
		DataDir:        dataDir,
	}
}

// detectLocalIP tries to determine the active outbound LAN IP address.
func detectLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ConfigPath returns the path to config.json inside dataDir.
func (c *Config) ConfigPath() string {
	return filepath.Join(c.DataDir, "config.json")
}

// Save persists the configuration to disk with secure 0600 permissions.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(c.DataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile := c.ConfigPath() + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write tmp config: %w", err)
	}

	if err := os.Rename(tmpFile, c.ConfigPath()); err != nil {
		return fmt.Errorf("failed to commit config file: %w", err)
	}
	return nil
}

// Load loads the configuration from dataDir.
func Load(dataDir string) (*Config, error) {
	cfgPath := filepath.Join(dataDir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(dataDir), ErrNotConfigured
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.DataDir = dataDir
	return &cfg, nil
}
