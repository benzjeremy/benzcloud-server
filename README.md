# ☁️ BenzCloud Server

> **Notice:** BenzCloud is in **active development / pre-release status** (`v1.0`). APIs, protocols, and interfaces are actively evolving.

BenzCloud Server is a privacy-first, local enterprise suite and open-source alternative to Microsoft 365 and Nextcloud written in pure Go. It features automatic zero-configuration Mesh-VPN networking (Slack Nebula), an authoritative Custom DNS server (RFC 1035), encrypted document storage, virtual host routing, and an extensible modular plugin architecture.

## 🚀 Key Features

- **One-Click Setup Wizard:** Enter your desired base domain (e.g. `benzjeremy.de` or `intern`) and admin password. The mesh VPN, custom DNS server, and routing configure themselves automatically without manual router port forwardings.
- **Zero-Config Mesh-VPN (Slack Nebula):** Seamless P2P overlay network across strict NATs and firewalls with out-of-the-box UDP hole-punching.
- **Authoritative Custom DNS Server:** Built-in lightweight RFC 1035 DNS server resolving system subdomains (`vpn.`, `drive.`, `mail.`, `chat.`) and custom web subdomains with fallback upstream forwarding.
- **Permanent VPN & DNS Privileges:** By architectural decree, network connectivity (VPN & DNS) is unconditionally active for all authenticated user accounts and cannot be revoked.
- **Encrypted Drive at Rest:** AES-256-GCM encrypted document vault with PBKDF2 (100,000 rounds) key derivation.
- **Modular Plugin Gateway:** Supervises and reverse-proxies modular enterprise micro-services (`benzcloud-plugin-web`, `benzcloud-plugin-mail`, `benzcloud-plugin-chat`).
- **Multi-Platform Support:** Native binaries for PC (Linux x86_64, Windows x86_64) and native Android APK.

---

## 📦 Installation & Usage

### Linux (x86_64)
```bash
# Run server daemon
./benzcloud-server -daemon -port 8080 -dns-port 53

# Or install via Go
go install github.com/benzjeremy/benzcloud-server@latest
```

### Windows (x86_64)
Launch `benzcloud-server.exe` to run the server daemon and open the management cockpit.

### Android
Install `benzcloud-server-v1.0.apk` to run BenzCloud as a mobile node or micro-server on Android.

---

## 👥 Authors & Credits
- **Jeremy Benz** ([@benzjeremy](https://github.com/benzjeremy)) – Lead Engineer & Project Creator
- Pair-programmed with AI Assistant (Google Antigravity)
- © 2026 Jeremy Benz

## 📄 License & Third-Party Notices
- **Main Project:** Released under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
- **Slack Nebula:** Mesh-VPN networking is powered by Slack Nebula, licensed under the [MIT License](https://github.com/slackhq/nebula/blob/master/LICENSE).
