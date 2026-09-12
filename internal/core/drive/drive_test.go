package drive

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/benzjeremy/benzcloud-server/internal/core/crypto"
)

func TestDriveManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_drive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	salt, _ := crypto.GenerateSalt(16)
	masterKey := crypto.DeriveKey("MasterSecretDriveKey2026", salt)

	dm, err := NewDriveManager(tempDir, masterKey)
	if err != nil {
		t.Fatalf("NewDriveManager failed: %v", err)
	}

	username := "jeremy"
	content := []byte("Strictly Confidential Cloud Document: Project BenzCloud Architecture")

	// 1. Save file
	item, err := dm.SaveFile(username, "docs/architecture.txt", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	if item.Name != "architecture.txt" {
		t.Fatalf("Expected file name architecture.txt, got %s", item.Name)
	}

	// Verify encryption on physical disk
	rawOnDisk, err := os.ReadFile(filepath.Join(tempDir, "drive", username, "docs", "architecture.txt"))
	if err != nil {
		t.Fatalf("Failed to read raw file from disk: %v", err)
	}
	if bytes.Contains(rawOnDisk, content) {
		t.Fatal("SECURITY BUG: Raw disk content contains unencrypted plaintext!")
	}

	// 2. Read and decrypt file
	decrypted, readItem, err := dm.ReadFile(username, "docs/architecture.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(content, decrypted) {
		t.Fatalf("Decrypted content mismatch: got %s, want %s", string(decrypted), string(content))
	}
	if readItem.SHA256 != item.SHA256 {
		t.Fatalf("SHA256 checksum mismatch: %s vs %s", readItem.SHA256, item.SHA256)
	}

	// 3. List directory
	files, err := dm.ListDirectory(username, "docs")
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file in docs, got %d", len(files))
	}

	// 4. Path traversal test
	_, _, err = dm.ReadFile(username, "../../etc/passwd")
	if err == nil {
		t.Fatal("SECURITY BUG: Path traversal did not return error!")
	}

	// 5. Delete file
	if err := dm.DeleteFile(username, "docs/architecture.txt"); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
	_, _, err = dm.ReadFile(username, "docs/architecture.txt")
	if err == nil {
		t.Fatal("Expected error reading deleted file, got nil")
	}
}
