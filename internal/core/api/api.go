package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/benzjeremy/benzcloud-server/internal/core/auth"
	"github.com/benzjeremy/benzcloud-server/internal/core/config"
	"github.com/benzjeremy/benzcloud-server/internal/core/dns"
	"github.com/benzjeremy/benzcloud-server/internal/core/drive"
	"github.com/benzjeremy/benzcloud-server/internal/core/nebula"
	"github.com/benzjeremy/benzcloud-server/internal/core/plugins"
)

// Server encapsulates the REST API and state references.
type Server struct {
	cfg       *config.Config
	authMgr   *auth.Manager
	dnsSrv    *dns.Server
	nebulaMgr *nebula.Manager
	driveMgr  *drive.DriveManager
	pluginMgr *plugins.Manager
	mux       *http.ServeMux
}

// NewServer creates a new API controller.
func NewServer(
	cfg *config.Config,
	authMgr *auth.Manager,
	dnsSrv *dns.Server,
	nebulaMgr *nebula.Manager,
	driveMgr *drive.DriveManager,
	pluginMgr *plugins.Manager,
) *Server {
	s := &Server{
		cfg:       cfg,
		authMgr:   authMgr,
		dnsSrv:    dnsSrv,
		nebulaMgr: nebulaMgr,
		driveMgr:  driveMgr,
		pluginMgr: pluginMgr,
		mux:       http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/setup", s.handleSetup)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/pair", s.handlePair)
	s.mux.HandleFunc("/api/users", s.handleUsers)
	s.mux.HandleFunc("/api/plugins", s.handlePlugins)
	s.mux.HandleFunc("/api/plugins/toggle", s.handlePluginToggle)
	s.mux.HandleFunc("/api/drive/files", s.handleDriveList)
	s.mux.HandleFunc("/api/drive/upload", s.handleDriveUpload)
	s.mux.HandleFunc("/api/drive/download", s.handleDriveDownload)
	s.mux.HandleFunc("/api/drive/folder", s.handleDriveFolder)
	s.mux.HandleFunc("/api/drive/delete", s.handleDriveDelete)
	s.mux.HandleFunc("/api/dns/logs", s.handleDNSLogs)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, status int, msg string) {
	s.jsonResponse(w, status, map[string]string{"error": msg})
}

func (s *Server) authenticate(r *http.Request) (*auth.User, error) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return s.authMgr.ValidateSession(token)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var peers []*nebula.PeerInfo
	if s.nebulaMgr != nil {
		peers = s.nebulaMgr.GetPeers()
	}

	var pluginList []*plugins.PluginStatus
	if s.pluginMgr != nil {
		pluginList = s.pluginMgr.ListPlugins()
	}

	res := map[string]interface{}{
		"system":           "benzcloud-server",
		"version":          "v1.0",
		"setup_completed":  s.cfg.SetupCompleted,
		"base_domain":      s.cfg.BaseDomain,
		"server_local_ip":  s.cfg.ServerLocalIP,
		"server_vpn_ip":    s.cfg.ServerVPNIP,
		"http_port":        s.cfg.HTTPPort,
		"dns_port":         s.cfg.DNSPort,
		"vpn_port":         s.cfg.VPNPort,
		"nebula_running":   s.nebulaMgr != nil && s.nebulaMgr.IsRunning(),
		"mesh_peers":       len(peers),
		"users_count":      s.authMgr.UserCount(),
		"plugins":          pluginList,
		"timestamp":        time.Now().UTC(),
	}
	s.jsonResponse(w, http.StatusOK, res)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.cfg.SetupCompleted {
		s.jsonError(w, http.StatusBadRequest, "Setup is already completed")
		return
	}

	var req struct {
		BaseDomain    string `json:"base_domain"`
		AdminUsername string `json:"admin_username"`
		AdminPassword string `json:"admin_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.BaseDomain) == "" || strings.TrimSpace(req.AdminUsername) == "" || len(req.AdminPassword) < 8 {
		s.jsonError(w, http.StatusBadRequest, "Base domain, admin username and password (>= 8 chars) required")
		return
	}

	cleanDomain := strings.ToLower(strings.Trim(req.BaseDomain, "."))
	s.cfg.BaseDomain = cleanDomain

	// 1. Create Admin user
	adminPerms := auth.Permissions{
		Admin: true,
		Drive: true,
		Mail:  true,
		Chat:  true,
		Web:   true,
		VPN:   true,
		DNS:   true,
	}
	adminUser, err := s.authMgr.CreateUser(req.AdminUsername, req.AdminPassword, adminPerms)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create admin: %v", err))
		return
	}

	// 2. Initialize Nebula PKI
	if err := s.nebulaMgr.InitPKI(s.cfg.ServerVPNIP, s.cfg.ServerLocalIP, s.cfg.VPNPort); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize Nebula PKI: %v", err))
		return
	}
	_ = s.nebulaMgr.StartController()

	// 3. Mark setup complete and save config
	s.cfg.SetupCompleted = true
	if err := s.cfg.Save(); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	// 4. Log admin in
	session, err := s.authMgr.Authenticate(req.AdminUsername, req.AdminPassword)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to authenticate new admin")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"message":       "Setup completed successfully",
		"base_domain":   cleanDomain,
		"admin_user":    adminUser.Username,
		"session_token": session.Token,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	session, err := s.authMgr.Authenticate(req.Username, req.Password)
	if err != nil {
		s.jsonError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	user, _ := s.authMgr.ValidateSession(session.Token)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token":       session.Token,
		"user":        user.Username,
		"permissions": user.Permissions,
		"overlay_ip":  user.OverlayIP,
		"expires_at":  session.ExpiresAt,
	})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	session, err := s.authMgr.Authenticate(req.Username, req.Password)
	if err != nil {
		s.jsonError(w, http.StatusUnauthorized, "Pairing failed: Invalid credentials")
		return
	}

	user, err := s.authMgr.ValidateSession(session.Token)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "User lookup failed")
		return
	}

	certPEM, keyPEM, configYAML, err := s.nebulaMgr.IssueClientCert(
		user.Username,
		user.OverlayIP,
		s.cfg.ServerLocalIP,
		s.cfg.VPNPort,
	)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to issue client mesh certificate: %v", err))
		return
	}

	// Pre-register peer in DNS if needed
	s.dnsSrv.RegisterSubdomain("peer-"+user.Username, user.OverlayIP)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":             "paired",
		"username":           user.Username,
		"base_domain":        s.cfg.BaseDomain,
		"overlay_ip":         user.OverlayIP,
		"server_vpn_ip":      s.cfg.ServerVPNIP,
		"cert_pem":           string(certPEM),
		"key_pem":            string(keyPEM),
		"client_config_yaml": string(configYAML),
		"session_token":      session.Token,
		"permissions":        user.Permissions,
	})
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Admin {
		s.jsonError(w, http.StatusForbidden, "Admin privileges required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		users := s.authMgr.ListUsers()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"users": users})
	case http.MethodPost:
		var req struct {
			Username    string           `json:"username"`
			Password    string           `json:"password"`
			Permissions auth.Permissions `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.jsonError(w, http.StatusBadRequest, "Invalid payload")
			return
		}
		user, err := s.authMgr.CreateUser(req.Username, req.Password, req.Permissions)
		if err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.jsonResponse(w, http.StatusCreated, user)
	case http.MethodPut:
		var req struct {
			Username    string           `json:"username"`
			Permissions auth.Permissions `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.jsonError(w, http.StatusBadRequest, "Invalid payload")
			return
		}
		if err := s.authMgr.UpdatePermissions(req.Username, req.Permissions); err != nil {
			s.jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]bool{"success": true})
	default:
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	_ = currentUser
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"plugins": s.pluginMgr.ListPlugins(),
	})
}

func (s *Server) handlePluginToggle(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Admin {
		s.jsonError(w, http.StatusForbidden, "Admin privileges required")
		return
	}

	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := s.pluginMgr.SetPluginEnabled(req.ID, req.Enabled); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDriveList(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Drive {
		s.jsonError(w, http.StatusForbidden, "Drive access denied")
		return
	}

	folder := r.URL.Query().Get("path")
	files, err := s.driveMgr.ListDirectory(currentUser.Username, folder)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"files": files})
}

func (s *Server) handleDriveUpload(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Drive {
		s.jsonError(w, http.StatusForbidden, "Drive access denied")
		return
	}

	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing path parameter")
		return
	}

	item, err := s.driveMgr.SaveFile(currentUser.Username, path, r.Body)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, item)
}

func (s *Server) handleDriveDownload(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Drive {
		s.jsonError(w, http.StatusForbidden, "Drive access denied")
		return
	}

	path := r.URL.Query().Get("path")
	data, item, err := s.driveMgr.ReadFile(currentUser.Username, path)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, "File not found")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", item.Name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleDriveFolder(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Drive {
		s.jsonError(w, http.StatusForbidden, "Drive access denied")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := s.driveMgr.CreateFolder(currentUser.Username, req.Path); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDriveDelete(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Drive {
		s.jsonError(w, http.StatusForbidden, "Drive access denied")
		return
	}

	path := r.URL.Query().Get("path")
	if err := s.driveMgr.DeleteFile(currentUser.Username, path); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDNSLogs(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.authenticate(r)
	if err != nil || !currentUser.Permissions.Admin {
		s.jsonError(w, http.StatusForbidden, "Admin privileges required")
		return
	}
	logs := s.dnsSrv.GetQueryLog()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"logs": logs})
}

// Ensure unused import warnings are resolved
var _ = io.EOF
