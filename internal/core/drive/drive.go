package drive

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/benzjeremy/benzcloud-server/internal/core/crypto"
)

// FileItem represents a file or folder in the user's cloud drive.
type FileItem struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"` // relative path from user root, e.g. "documents/invoice.pdf"
	Size      int64     `json:"size"`
	IsDir     bool      `json:"is_dir"`
	ModTime   time.Time `json:"mod_time"`
	SHA256    string    `json:"sha256,omitempty"`
}

type DriveManager struct {
	baseDir   string
	masterKey []byte
	mu        sync.RWMutex
}

var (
	ErrInvalidPath  = errors.New("invalid or illegal file path")
	ErrFileNotFound = errors.New("file or directory not found")
)

// NewDriveManager creates a file manager with encrypted storage at rest.
func NewDriveManager(baseDir string, masterKey []byte) (*DriveManager, error) {
	drivePath := filepath.Join(baseDir, "drive")
	if err := os.MkdirAll(drivePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create drive directory: %w", err)
	}
	return &DriveManager{
		baseDir:   drivePath,
		masterKey: masterKey,
	}, nil
}

func (dm *DriveManager) userDir(username string) string {
	safeUser := filepath.Clean(username)
	safeUser = strings.ReplaceAll(safeUser, "/", "_")
	safeUser = strings.ReplaceAll(safeUser, "\\", "_")
	return filepath.Join(dm.baseDir, safeUser)
}

func (dm *DriveManager) resolvePath(username, relPath string) (string, error) {
	uDir := dm.userDir(username)
	cleanRel := filepath.Clean(strings.TrimPrefix(relPath, "/"))
	if strings.HasPrefix(cleanRel, "..") || strings.Contains(cleanRel, "/../") {
		return "", ErrInvalidPath
	}
	target := filepath.Join(uDir, cleanRel)
	if !strings.HasPrefix(target, uDir) {
		return "", ErrInvalidPath
	}
	return target, nil
}

// ListDirectory returns the list of files and folders in the specified folder.
func (dm *DriveManager) ListDirectory(username, folderPath string) ([]FileItem, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	target, err := dm.resolvePath(username, folderPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(target, 0700); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	var items []FileItem
	uDir := dm.userDir(username)

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(target, e.Name())
		rel, _ := filepath.Rel(uDir, full)

		items = append(items, FileItem{
			Name:    e.Name(),
			Path:    filepath.ToSlash(rel),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().UTC(),
		})
	}
	return items, nil
}

// SaveFile saves plaintext content to encrypted storage on disk.
func (dm *DriveManager) SaveFile(username, relPath string, r io.Reader) (*FileItem, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	target, err := dm.resolvePath(username, relPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return nil, err
	}

	rawBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	hasher.Write(rawBytes)
	hashHex := hex.EncodeToString(hasher.Sum(nil))

	// Encrypt at rest using AES-256-GCM
	encrypted, err := crypto.Encrypt(dm.masterKey, rawBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt file payload: %w", err)
	}

	if err := os.WriteFile(target, encrypted, 0600); err != nil {
		return nil, err
	}

	uDir := dm.userDir(username)
	rel, _ := filepath.Rel(uDir, target)

	return &FileItem{
		Name:    filepath.Base(target),
		Path:    filepath.ToSlash(rel),
		Size:    int64(len(rawBytes)),
		IsDir:   false,
		ModTime: time.Now().UTC(),
		SHA256:  hashHex,
	}, nil
}

// ReadFile decrypts and returns the content of the file.
func (dm *DriveManager) ReadFile(username, relPath string) ([]byte, *FileItem, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	target, err := dm.resolvePath(username, relPath)
	if err != nil {
		return nil, nil, err
	}

	encrypted, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrFileNotFound
		}
		return nil, nil, err
	}

	decrypted, err := crypto.Decrypt(dm.masterKey, encrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt file at rest: %w", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, nil, err
	}

	hasher := sha256.New()
	hasher.Write(decrypted)
	hashHex := hex.EncodeToString(hasher.Sum(nil))

	uDir := dm.userDir(username)
	rel, _ := filepath.Rel(uDir, target)

	item := &FileItem{
		Name:    filepath.Base(target),
		Path:    filepath.ToSlash(rel),
		Size:    int64(len(decrypted)),
		IsDir:   false,
		ModTime: info.ModTime().UTC(),
		SHA256:  hashHex,
	}

	return decrypted, item, nil
}

// DeleteFile removes a file or directory.
func (dm *DriveManager) DeleteFile(username, relPath string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	target, err := dm.resolvePath(username, relPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(target)
}

// CreateFolder creates a directory.
func (dm *DriveManager) CreateFolder(username, relPath string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	target, err := dm.resolvePath(username, relPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(target, 0700)
}
