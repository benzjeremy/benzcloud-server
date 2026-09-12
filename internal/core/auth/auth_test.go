package auth

import (
	"os"
	"testing"
)

func TestAuthManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_auth_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Create admin user
	adminPerms := Permissions{Admin: true, Drive: true, Mail: true, Chat: true, Web: true}
	admin, err := mgr.CreateUser("admin", "AdminSecret123!", adminPerms)
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}
	if !admin.Permissions.VPN || !admin.Permissions.DNS {
		t.Fatal("Admin must have VPN and DNS active")
	}
	if admin.OverlayIP != "10.42.0.2" {
		t.Fatalf("Expected admin overlay IP 10.42.0.2, got %s", admin.OverlayIP)
	}

	// 2. Authenticate
	sess, err := mgr.Authenticate("admin", "AdminSecret123!")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if len(sess.Token) != 64 {
		t.Fatalf("Expected 64-char token, got %d", len(sess.Token))
	}

	// 3. Validate Session
	valUser, err := mgr.ValidateSession(sess.Token)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if valUser.Username != "admin" {
		t.Fatalf("Expected username admin, got %s", valUser.Username)
	}

	// 4. Create regular user and attempt to revoke VPN/DNS
	restrictedPerms := Permissions{
		Admin: false,
		Drive: false,
		Mail:  false,
		Chat:  false,
		Web:   false,
		VPN:   false, // Try to revoke
		DNS:   false, // Try to revoke
	}
	user2, err := mgr.CreateUser("worker", "WorkerSecret123!", restrictedPerms)
	if err != nil {
		t.Fatalf("CreateUser worker failed: %v", err)
	}
	// Verify invariant: VPN & DNS MUST remain true!
	if !user2.Permissions.VPN || !user2.Permissions.DNS {
		t.Fatal("CRITICAL INVARIANT VIOLATED: VPN and DNS permissions must remain true even when requested false")
	}

	// 5. Try updating permissions to revoke VPN/DNS
	err = mgr.UpdatePermissions("worker", Permissions{VPN: false, DNS: false})
	if err != nil {
		t.Fatalf("UpdatePermissions failed: %v", err)
	}
	workerUpdated, _ := mgr.ValidateSession("")
	_ = workerUpdated
	list := mgr.ListUsers()
	for _, u := range list {
		if u.Username == "worker" {
			if !u.Permissions.VPN || !u.Permissions.DNS {
				t.Fatal("CRITICAL INVARIANT VIOLATED: VPN and DNS permissions must remain true after UpdatePermissions")
			}
		}
	}

	// 6. Reload from disk and verify persistence
	mgr2, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Reloading manager failed: %v", err)
	}
	if mgr2.UserCount() != 2 {
		t.Fatalf("Expected 2 users after reload, got %d", mgr2.UserCount())
	}
}
