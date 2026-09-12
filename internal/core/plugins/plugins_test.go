package plugins

import (
	"os"
	"testing"
)

func TestPluginManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_plugins_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir, "benzjeremy.de", "SecretServerToken32BytesHexToken2026")

	// 1. Initial list
	list := mgr.ListPlugins()
	if len(list) != 3 {
		t.Fatalf("Expected 3 standard plugins, got %d", len(list))
	}

	// 2. Start virtual plugin
	if err := mgr.StartPlugin("mail"); err != nil {
		t.Fatalf("StartPlugin mail failed: %v", err)
	}

	// 3. Subdomain routing
	port, ok := mgr.GetTargetForSubdomain("mail")
	if !ok || port != 8092 {
		t.Fatalf("Expected port 8092 for mail subdomain, got %d (ok=%v)", port, ok)
	}

	portChat, ok := mgr.GetTargetForSubdomain("chat")
	if !ok || portChat != 8093 {
		t.Fatalf("Expected port 8093 for chat subdomain, got %d (ok=%v)", portChat, ok)
	}

	// Custom subdomain routing to web plugin
	portCustom, ok := mgr.GetTargetForSubdomain("my-custom-blog")
	if !ok || portCustom != 8091 {
		t.Fatalf("Expected port 8091 for custom web subdomain, got %d (ok=%v)", portCustom, ok)
	}

	// 4. Disable and verify
	if err := mgr.SetPluginEnabled("chat", false); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	_, ok = mgr.GetTargetForSubdomain("chat")
	if ok {
		t.Fatal("Disabled plugin should not resolve subdomain")
	}
}
