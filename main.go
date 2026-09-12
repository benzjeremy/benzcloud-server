package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/benzjeremy/benzcloud-server/internal/core/api"
	"github.com/benzjeremy/benzcloud-server/internal/core/auth"
	"github.com/benzjeremy/benzcloud-server/internal/core/config"
	"github.com/benzjeremy/benzcloud-server/internal/core/crypto"
	"github.com/benzjeremy/benzcloud-server/internal/core/dns"
	"github.com/benzjeremy/benzcloud-server/internal/core/drive"
	"github.com/benzjeremy/benzcloud-server/internal/core/nebula"
	"github.com/benzjeremy/benzcloud-server/internal/core/plugins"
	"github.com/benzjeremy/benzcloud-server/internal/core/proxy"
)

//go:embed web/*
var webFS embed.FS

const Version = "v1.0"

func main() {
	var (
		portFlag    int
		dnsPortFlag int
		dataDirFlag string
		domainFlag  string
		daemonFlag  bool
		versionFlag bool
	)

	flag.IntVar(&portFlag, "port", 8080, "HTTP server port (default 8080)")
	flag.IntVar(&dnsPortFlag, "dns-port", 53, "DNS server port (default 53)")
	flag.StringVar(&dataDirFlag, "data", "", "Data directory for configs, files, and certificates")
	flag.StringVar(&domainFlag, "domain", "", "Base domain override")
	flag.BoolVar(&daemonFlag, "daemon", false, "Run in background daemon mode without desktop window")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.Parse()

	if versionFlag {
		fmt.Printf("BenzCloud Server %s (Lead Engineer: Jeremy Benz • GNU GPLv3)\n", Version)
		return
	}

	if dataDirFlag == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			dataDirFlag = "./benzcloud-data"
		} else {
			dataDirFlag = filepath.Join(home, ".benzcloud", "server")
		}
	}

	if err := os.MkdirAll(dataDirFlag, 0700); err != nil {
		log.Fatalf("Fatal: could not create data dir %s: %v\n", dataDirFlag, err)
	}

	// 1. Config
	cfg, err := config.Load(dataDirFlag)
	if err != nil && err != config.ErrNotConfigured {
		log.Fatalf("Fatal: failed to load config: %v\n", err)
	}
	if portFlag != 8080 {
		cfg.HTTPPort = portFlag
	}
	if dnsPortFlag != 53 {
		cfg.DNSPort = dnsPortFlag
	}
	if domainFlag != "" {
		cfg.BaseDomain = domainFlag
	}

	// 2. Master Encryption Key & Server Token
	if cfg.MasterSalt == "" {
		saltBytes, _ := crypto.GenerateSalt(16)
		cfg.MasterSalt = fmt.Sprintf("%x", saltBytes)
	}
	if cfg.ServerToken == "" {
		tok, _ := crypto.GenerateToken()
		cfg.ServerToken = tok
	}
	_ = cfg.Save()

	masterSalt, _ := crypto.GenerateSalt(16)
	masterKey := crypto.DeriveKey(cfg.ServerToken, masterSalt)

	// 3. Auth Manager
	authMgr, err := auth.NewManager(dataDirFlag)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize auth manager: %v\n", err)
	}

	// 4. DNS Server
	dnsSrv := dns.NewServer(cfg.BaseDomain, cfg.ServerVPNIP, cfg.DNSPort)
	if err := dnsSrv.Start(); err != nil {
		log.Printf("[BenzCloud DNS] Notice: Port %d requires root/CAP_NET_BIND_SERVICE (%v). Trying unprivileged fallback port...\n", cfg.DNSPort, err)
		fallbackPorts := []int{1053, 5354, 8053}
		started := false
		for _, fbPort := range fallbackPorts {
			candidate := dns.NewServer(cfg.BaseDomain, cfg.ServerVPNIP, fbPort)
			if err := candidate.Start(); err == nil {
				dnsSrv = candidate
				cfg.DNSPort = fbPort
				started = true
				break
			}
		}
		if !started {
			log.Printf("[BenzCloud DNS] Warning: Could not bind fallback DNS server. Local DNS resolution will be disabled.\n")
		}
	}

	// 5. Nebula Mesh VPN Manager
	nebulaMgr := nebula.NewManager(dataDirFlag)
	if cfg.SetupCompleted {
		_ = nebulaMgr.InitPKI(cfg.ServerVPNIP, cfg.ServerLocalIP, cfg.VPNPort)
		_ = nebulaMgr.StartController()
	}

	// 6. Drive Manager
	driveMgr, err := drive.NewDriveManager(dataDirFlag, masterKey)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize Drive manager: %v\n", err)
	}

	// 7. Plugin Manager
	pluginMgr := plugins.NewManager(dataDirFlag, cfg.BaseDomain, cfg.ServerToken)
	_ = pluginMgr.StartPlugin("web")
	_ = pluginMgr.StartPlugin("mail")
	_ = pluginMgr.StartPlugin("chat")

	// 8. REST API & Static Web UI
	apiServer := api.NewServer(cfg, authMgr, dnsSrv, nebulaMgr, driveMgr, pluginMgr)

	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Fatal: failed to extract web assets: %v\n", err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	mainMux := http.NewServeMux()
	mainMux.Handle("/api/", apiServer.Handler())
	mainMux.Handle("/", fileServer)

	// 9. Drive Subdomain Handler
	driveMux := http.NewServeMux()
	driveMux.Handle("/api/", apiServer.Handler())
	driveMux.Handle("/", fileServer)

	// 10. Virtual Host Reverse Proxy
	router := proxy.NewRouter(cfg.BaseDomain, cfg.ServerLocalIP, cfg.ServerToken, pluginMgr, mainMux, driveMux)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("🚀 [BenzCloud Server %s] Running at: http://127.0.0.1:%d / http://%s:%d\n",
			Version, cfg.HTTPPort, cfg.ServerLocalIP, cfg.HTTPPort)
		log.Printf("🌐 [BenzCloud DNS] Base Domain: %s (System subdomains: vpn, drive, mail, chat)\n", cfg.BaseDomain)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fatal: HTTP server failed: %v\n", err)
		}
	}()

	appURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.HTTPPort)

	// If interactive desktop mode
	if !daemonFlag && os.Getenv("DISPLAY") != "" && os.Getenv("HEADLESS") != "1" {
		go func() {
			time.Sleep(200 * time.Millisecond)
			LaunchGUI("BenzCloud – Micro-Enterprise Suite", appURL, 1180, 800)
		}()
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\nShutting down BenzCloud Server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpServer.Shutdown(ctx)
	_ = dnsSrv.Stop()
	_ = nebulaMgr.Stop()
	_ = pluginMgr.StopPlugin("web")
	_ = pluginMgr.StopPlugin("mail")
	_ = pluginMgr.StopPlugin("chat")

	log.Println("BenzCloud Server safely terminated.")
}
