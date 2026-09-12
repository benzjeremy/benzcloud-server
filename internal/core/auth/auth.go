package auth

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benzjeremy/benzcloud-server/internal/core/crypto"
)

// Permissions defines user feature rights.
// NOTE: VPN and DNS are ALWAYS active and cannot be revoked!
type Permissions struct {
	Admin bool `json:"admin"`
	Drive bool `json:"drive"`
	Mail  bool `json:"mail"`
	Chat  bool `json:"chat"`
	Web   bool `json:"web"`
	VPN   bool `json:"vpn"` // Always true
	DNS   bool `json:"dns"` // Always true
}

// User represents a user account in benzcloud.
type User struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	PasswordHash string      `json:"password_hash"`
	Salt         string      `json:"salt"`
	OverlayIP    string      `json:"overlay_ip"` // e.g. 10.42.0.2
	Permissions  Permissions `json:"permissions"`
	CreatedAt    time.Time   `json:"created_at"`
	LastLogin    time.Time   `json:"last_login"`
}

// HasVPN returns true always. VPN cannot be revoked.
func (u *User) HasVPN() bool {
	return true
}

// HasDNS returns true always. DNS cannot be revoked.
func (u *User) HasDNS() bool {
	return true
}

type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Manager struct {
	dataDir  string
	users    map[string]*User    // keyed by username
	sessions map[string]*Session // keyed by token
	nextIP   int                 // counter for 10.42.0.X
	mu       sync.RWMutex
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrSessionExpired    = errors.New("session expired or invalid")
)

// NewManager initializes the auth store.
func NewManager(dataDir string) (*Manager, error) {
	m := &Manager{
		dataDir:  dataDir,
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
		nextIP:   2, // 1 is server (10.42.0.1)
	}

	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load user database: %w", err)
	}
	return m, nil
}

func (m *Manager) dbPath() string {
	return filepath.Join(m.dataDir, "users.json")
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.dbPath())
	if err != nil {
		return err
	}

	var stored struct {
		Users  []*User `json:"users"`
		NextIP int     `json:"next_ip"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	for _, u := range stored.Users {
		// Enforce invariant: VPN and DNS are always true
		u.Permissions.VPN = true
		u.Permissions.DNS = true
		m.users[u.Username] = u
	}
	if stored.NextIP > m.nextIP {
		m.nextIP = stored.NextIP
	}
	return nil
}

func (m *Manager) save() error {
	var userList []*User
	for _, u := range m.users {
		userList = append(userList, u)
	}

	payload := struct {
		Users  []*User `json:"users"`
		NextIP int     `json:"next_ip"`
	}{
		Users:  userList,
		NextIP: m.nextIP,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := m.dbPath() + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, m.dbPath())
}

// CreateUser creates a new user with PBKDF2-hashed password and assigns a Nebula Overlay IP.
// The permissions for VPN and DNS will ALWAYS be set to true.
func (m *Manager) CreateUser(username, password string, perms Permissions) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[username]; exists {
		return nil, ErrUserAlreadyExists
	}

	saltBytes, err := crypto.GenerateSalt(16)
	if err != nil {
		return nil, err
	}

	key := crypto.DeriveKey(password, saltBytes)
	hashHex := hex.EncodeToString(key)
	saltHex := hex.EncodeToString(saltBytes)

	// Invariant: VPN and DNS cannot be revoked
	perms.VPN = true
	perms.DNS = true

	overlayIP := fmt.Sprintf("10.42.0.%d", m.nextIP)
	m.nextIP++

	user := &User{
		ID:           fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Username:     username,
		PasswordHash: hashHex,
		Salt:         saltHex,
		OverlayIP:    overlayIP,
		Permissions:  perms,
		CreatedAt:    time.Now().UTC(),
	}

	m.users[username] = user
	if err := m.save(); err != nil {
		return nil, err
	}
	return user, nil
}

// Authenticate verifies password and returns a session token.
func (m *Manager) Authenticate(username, password string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	saltBytes, err := hex.DecodeString(user.Salt)
	if err != nil {
		return nil, errors.New("corrupted salt")
	}

	expectedHash, err := hex.DecodeString(user.PasswordHash)
	if err != nil {
		return nil, errors.New("corrupted hash")
	}

	calculatedHash := crypto.DeriveKey(password, saltBytes)
	if !bytes.Equal(expectedHash, calculatedHash) {
		return nil, ErrInvalidPassword
	}

	user.LastLogin = time.Now().UTC()
	_ = m.save()

	token, err := crypto.GenerateToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
	}

	m.sessions[token] = session
	return session, nil
}

// ValidateSession validates the token and returns the corresponding user.
func (m *Manager) ValidateSession(token string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[token]
	if !exists || time.Now().UTC().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	user, exists := m.users[session.Username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// ListUsers returns a safe list of all users without sensitive hash/salt.
func (m *Manager) ListUsers() []*User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*User
	for _, u := range m.users {
		safeCopy := *u
		safeCopy.PasswordHash = ""
		safeCopy.Salt = ""
		// Invariant holds
		safeCopy.Permissions.VPN = true
		safeCopy.Permissions.DNS = true
		list = append(list, &safeCopy)
	}
	return list
}

// UpdatePermissions updates user permissions, strictly preserving VPN and DNS.
func (m *Manager) UpdatePermissions(username string, perms Permissions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[username]
	if !exists {
		return ErrUserNotFound
	}

	// Always enforce: VPN and DNS are immutable
	perms.VPN = true
	perms.DNS = true
	user.Permissions = perms

	return m.save()
}

// UserCount returns total count of registered users.
func (m *Manager) UserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}
