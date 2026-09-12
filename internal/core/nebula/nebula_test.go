package nebula

import (
	"os"
	"strings"
	"testing"
)

func TestNebulaManagerPKI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_nebula_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir)

	// 1. Initialize PKI
	serverOverlayIP := "10.42.0.1"
	serverLocalIP := "192.168.1.100"
	vpnPort := 4242

	if err := mgr.InitPKI(serverOverlayIP, serverLocalIP, vpnPort); err != nil {
		t.Fatalf("InitPKI failed: %v", err)
	}

	// Verify CA files created
	if _, err := os.Stat(mgr.caCertPath); err != nil {
		t.Fatalf("CA cert missing: %v", err)
	}
	if _, err := os.Stat(mgr.serverConfigPath); err != nil {
		t.Fatalf("Server config YAML missing: %v", err)
	}

	// 2. Issue client certificate for a user
	certPEM, keyPEM, configYAML, err := mgr.IssueClientCert("worker-bob", "10.42.0.2", serverLocalIP, vpnPort)
	if err != nil {
		t.Fatalf("IssueClientCert failed: %v", err)
	}

	if len(certPEM) == 0 || len(keyPEM) == 0 || len(configYAML) == 0 {
		t.Fatal("Expected non-empty cert, key and config YAML")
	}

	cfgStr := string(configYAML)
	if !strings.Contains(cfgStr, "10.42.0.2") {
		t.Fatal("Client config must contain assigned overlay IP")
	}
	if !strings.Contains(cfgStr, serverLocalIP) {
		t.Fatal("Client config must contain server endpoint IP")
	}

	// 3. Verify peers list
	peers := mgr.GetPeers()
	if len(peers) != 2 {
		t.Fatalf("Expected 2 peers (server + client), got %d", len(peers))
	}

	// 4. Start controller and verify state
	if err := mgr.StartController(); err != nil {
		t.Fatalf("StartController failed: %v", err)
	}
	if !mgr.IsRunning() {
		t.Fatal("Expected IsRunning to be true")
	}
	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if mgr.IsRunning() {
		t.Fatal("Expected IsRunning to be false after stop")
	}
}
