// BenzCloud Frontend Application Engine
// Author: Jeremy Benz (@benzjeremy) • GNU GPL-3.0

let currentLang = localStorage.getItem("benzcloud_lang") || "de";
let sessionToken = localStorage.getItem("benzcloud_token") || "";
let systemConfig = null;
let currentPath = "";

const i18n = {
  de: {
    setup_badge: "✨ One-Click Ersteinrichtung",
    setup_title: "Willkommen bei BenzCloud",
    setup_desc: "Deine datenschutzfreundliche, lokale Enterprise-Suite. Trage deine Wunsch-Domain ein – Mesh-VPN, DNS-Server und Routing konfigurieren sich vollautomatisch ohne Router-Portfreigaben.",
    lbl_domain: "Wunsch-Domain (Basis für System & Dienste):",
    hint_domain: "Subdomains wie vpn, drive und mail werden automatisch erzeugt.",
    lbl_admin_user: "Administrator-Benutzername:",
    lbl_admin_pass: "Administrator-Passwort (mind. 8 Zeichen):",
    sec_title: "Zero-Dummy-Security ab Werk:",
    sec_desc: "PBKDF2 Schlüsselableitung (100.000 Runden), AES-256-GCM Verschlüsselung at Rest und gegenseitig authentifiziertes Slack Nebula P2P-Mesh.",
    btn_start_setup: "🚀 BenzCloud jetzt automatisch einrichten",
    login_title: "BenzCloud Anmeldung",
    login_desc: "Melde dich mit deinem Konto an, um auf das Cockpit und deine Dienste zuzugreifen.",
    lbl_username: "Benutzername:",
    lbl_password: "Passwort:",
    btn_login: "Anmelden",
    logout: "Abmelden",
    m_domain: "Basis-Domain",
    m_vpn: "Mesh-VPN (Nebula)",
    m_dns: "Custom DNS Server",
    m_peers: "Verbundene Peers",
    m_peers_sub: "P2P Mesh Nodes",
    tab_overview: "🧭 Schnellstart",
    tab_drive: "📁 Drive (Cloud-Dateien)",
    tab_vpn: "🛡️ Mesh-VPN & Peers",
    tab_dns: "🌐 DNS & Routing",
    tab_plugins: "🧩 Plugins",
    tab_users: "👥 Benutzerverwaltung",
    services_title: "Verfügbare Enterprise-Dienste",
    srv_drive: "BenzCloud Drive",
    srv_drive_desc: "Verschlüsselter Dateispeicher, Dokumentenablage & Dateifreigabe (Nextcloud-Alternative).",
    srv_mail: "BenzCloud Mail",
    srv_mail_desc: "Geschlossenes internes E-Mail-System (SMTP/IMAP) mit integrierter Webmail-Oberfläche.",
    srv_chat: "BenzCloud Chat",
    srv_chat_desc: "Echtzeit-Team-Chat (Slack/Teams-Alternative) für verschlüsselte Kommunikation.",
    srv_web: "Web-Hosting Engine",
    srv_web_desc: "Eigene Webseiten unter frei wählbaren Subdomains (HTML, PHP, Astro).",
    pair_title: "Client-Kopplung (Android & PC)",
    pair_desc: "Verbinde die Client-App einmalig über die lokale Server-IP. Danach läuft die gesamte Kommunikation automatisch verschlüsselt über das Mesh-VPN und den Custom-DNS.",
    pair_server_endpoint: "Server LAN-Kopplungs-Endpunkt:",
    pair_cli_cmd: "Verbindungsbefehl (BenzCloud-Client CLI):",
    drive_vault_title: "Verschlüsselter Dateispeicher (AES-256-GCM)",
    btn_upload: "⬆️ Datei hochladen",
    btn_new_folder: "📁 Neuer Ordner",
    th_name: "Name",
    th_size: "Größe",
    th_date: "Änderungsdatum",
    th_actions: "Aktionen",
    loading_files: "Lade Dateien...",
    vpn_title: "Slack Nebula Mesh-VPN & Peer-Topologie",
    vpn_desc: "Das P2P Mesh-Overlay verbindet alle autorisierten Endgeräte verschlüsselt ohne Portweiterleitungen am Router.",
    th_peer_name: "Node / Peer Name",
    th_overlay_ip: "Overlay IP",
    th_role: "Rolle",
    th_status: "Status",
    th_valid_until: "Zertifikat gültig bis",
    dns_title: "Custom DNS Server & Subdomain-Routing",
    dns_records_title: "Autoritative System-Subdomains",
    dns_logs_title: "Live DNS-Abfragen (RFC 1035)",
    plugins_title: "Modulare Enterprise-Plugins",
    plugins_desc: "Der Server orchestriert Module in getrennten Prozessen über token-gesicherte Schnittstellen.",
    users_title: "Benutzer- und Rechteverwaltung",
    btn_create_user: "➕ Neuer Benutzer",
    th_user: "Benutzer",
    th_perms: "Berechtigungen",
    th_base_perms: "Basis-Privilegien (Fixiert)",
    modal_create_user_title: "Neuen Benutzer anlegen",
    immutable_perms_text: "VPN & DNS sind unveränderlich aktiv und können nicht entzogen werden.",
    btn_cancel: "Abbrechen",
    btn_save: "Erstellen",
    pre_release_pill: "Pre-Release / In aktiver Entwicklung",
    btn_toggle_on: "Aktiviert",
    btn_toggle_off: "Deaktiviert",
    btn_download: "⬇️ Herunterladen",
    btn_delete: "🗑️ Löschen",
    prompt_folder_name: "Name des neuen Ordners:",
    confirm_delete: "Möchtest du diese Datei wirklich löschen?"
  },
  en: {
    setup_badge: "✨ One-Click Initial Setup",
    setup_title: "Welcome to BenzCloud",
    setup_desc: "Your privacy-first, local enterprise suite. Enter your desired domain – Mesh-VPN, DNS server, and routing configure automatically without router port forwards.",
    lbl_domain: "Target Domain (Base for system & services):",
    hint_domain: "Subdomains like vpn, drive, and mail will be generated automatically.",
    lbl_admin_user: "Administrator Username:",
    lbl_admin_pass: "Administrator Password (min. 8 characters):",
    sec_title: "Zero-Dummy-Security by Design:",
    sec_desc: "PBKDF2 key derivation (100,000 rounds), AES-256-GCM encryption at rest, and mutually authenticated Slack Nebula P2P mesh.",
    btn_start_setup: "🚀 Automatically configure BenzCloud now",
    login_title: "BenzCloud Authentication",
    login_desc: "Sign in with your account to access your cockpit and services.",
    lbl_username: "Username:",
    lbl_password: "Password:",
    btn_login: "Sign In",
    logout: "Log Out",
    m_domain: "Base Domain",
    m_vpn: "Mesh-VPN (Nebula)",
    m_dns: "Custom DNS Server",
    m_peers: "Connected Peers",
    m_peers_sub: "P2P Mesh Nodes",
    tab_overview: "🧭 Quickstart",
    tab_drive: "📁 Drive (Cloud Files)",
    tab_vpn: "🛡️ Mesh-VPN & Peers",
    tab_dns: "🌐 DNS & Routing",
    tab_plugins: "🧩 Plugins",
    tab_users: "👥 User Management",
    services_title: "Available Enterprise Services",
    srv_drive: "BenzCloud Drive",
    srv_drive_desc: "Encrypted file vault, document store & file sharing (Nextcloud alternative).",
    srv_mail: "BenzCloud Mail",
    srv_mail_desc: "Closed internal email system (SMTP/IMAP) with integrated webmail client.",
    srv_chat: "BenzCloud Chat",
    srv_chat_desc: "Real-time team chat (Slack/Teams alternative) for direct communication.",
    srv_web: "Web-Hosting Engine",
    srv_web_desc: "Deploy custom websites under arbitrary subdomains (HTML, PHP, Astro).",
    pair_title: "Client Pairing (Android & PC)",
    pair_desc: "Connect your client app once via local server IP. Afterward, all traffic automatically flows securely through Mesh-VPN and custom DNS.",
    pair_server_endpoint: "Server LAN Pairing Endpoint:",
    pair_cli_cmd: "Connect Command (BenzCloud-Client CLI):",
    drive_vault_title: "Encrypted File Storage (AES-256-GCM)",
    btn_upload: "⬆️ Upload File",
    btn_new_folder: "📁 New Folder",
    th_name: "Name",
    th_size: "Size",
    th_date: "Modified Date",
    th_actions: "Actions",
    loading_files: "Loading files...",
    vpn_title: "Slack Nebula Mesh-VPN & Peer Topology",
    vpn_desc: "The P2P mesh overlay connects all authorized endpoints securely without router port forwarding.",
    th_peer_name: "Node / Peer Name",
    th_overlay_ip: "Overlay IP",
    th_role: "Role",
    th_status: "Status",
    th_valid_until: "Certificate Valid Until",
    dns_title: "Custom DNS Server & Subdomain Routing",
    dns_records_title: "Authoritative System Subdomains",
    dns_logs_title: "Live DNS Queries (RFC 1035)",
    plugins_title: "Modular Enterprise Plugins",
    plugins_desc: "The server orchestrates modules in separate processes across token-authenticated interfaces.",
    users_title: "User & Rights Management",
    btn_create_user: "➕ New User",
    th_user: "User",
    th_perms: "Permissions",
    th_base_perms: "Base Privileges (Locked)",
    modal_create_user_title: "Create New User",
    immutable_perms_text: "VPN & DNS are permanently active and cannot be revoked.",
    btn_cancel: "Cancel",
    btn_save: "Create",
    pre_release_pill: "Pre-Release / In Active Development",
    btn_toggle_on: "Enabled",
    btn_toggle_off: "Disabled",
    btn_download: "⬇️ Download",
    btn_delete: "🗑️ Delete",
    prompt_folder_name: "Name of new folder:",
    confirm_delete: "Are you sure you want to delete this file?"
  }
};

document.addEventListener("DOMContentLoaded", () => {
  applyLanguage(currentLang);
  checkSystemStatus();
  setInterval(refreshActiveTabData, 3000);
});

function setLanguage(lang) {
  currentLang = lang;
  localStorage.setItem("benzcloud_lang", lang);
  applyLanguage(lang);
}

function applyLanguage(lang) {
  document.documentElement.lang = lang;
  document.querySelectorAll("[data-i18n]").forEach(el => {
    const key = el.getAttribute("data-i18n");
    if (i18n[lang] && i18n[lang][key]) {
      el.textContent = i18n[lang][key];
    }
  });
  document.getElementById("langDE").classList.toggle("active", lang === "de");
  document.getElementById("langEN").classList.toggle("active", lang === "en");
}

async function checkSystemStatus() {
  try {
    const res = await fetch("/api/status");
    const data = await res.json();
    systemConfig = data;

    document.getElementById("headerDomainPill").textContent = data.base_domain || "intern";
    document.getElementById("mDomain").textContent = data.base_domain || "-";
    document.getElementById("mLocalIP").textContent = "LAN: " + (data.server_local_ip || "127.0.0.1");
    document.getElementById("mVPNIP").textContent = "Overlay IP: " + (data.server_vpn_ip || "10.42.0.1");
    document.getElementById("mDNSPort").textContent = `Port ${data.dns_port} (UDP & TCP)`;
    document.getElementById("mPeers").textContent = data.mesh_peers || "0";

    // Update pairing links
    const pairUrl = `http://${data.server_local_ip}:${data.http_port}/api/pair`;
    document.getElementById("pairEndpoint").textContent = pairUrl;
    document.getElementById("pairCliCmd").textContent = `benzcloud-client pair -server http://${data.server_local_ip}:${data.http_port} -user admin`;

    // Service URLs
    document.getElementById("urlDrive").textContent = `http://drive.${data.base_domain}`;
    document.getElementById("linkDrive").href = `http://drive.${data.base_domain}`;
    document.getElementById("urlMail").textContent = `http://mail.${data.base_domain}`;
    document.getElementById("linkMail").href = `http://mail.${data.base_domain}`;
    document.getElementById("urlChat").textContent = `http://chat.${data.base_domain}`;
    document.getElementById("linkChat").href = `http://chat.${data.base_domain}`;
    document.getElementById("urlWeb").textContent = `http://*.${data.base_domain}`;
    document.getElementById("linkWeb").href = `http://${data.base_domain}`;

    if (!data.setup_completed) {
      showView("setupWizard");
      return;
    }

    if (!sessionToken) {
      showView("loginView");
      return;
    }

    showView("dashboardView");
    document.getElementById("btnLogout").style.display = "block";
    loadDashboardData();
  } catch (err) {
    console.error("Failed to fetch system status:", err);
  }
}

function showView(viewId) {
  document.getElementById("setupWizard").style.display = viewId === "setupWizard" ? "block" : "none";
  document.getElementById("loginView").style.display = viewId === "loginView" ? "block" : "none";
  document.getElementById("dashboardView").style.display = viewId === "dashboardView" ? "block" : "none";
}

async function submitSetup(e) {
  e.preventDefault();
  const baseDomain = document.getElementById("setupDomain").value.trim();
  const adminUser = document.getElementById("setupAdminUser").value.trim();
  const adminPass = document.getElementById("setupAdminPass").value;

  const btn = document.getElementById("btnSubmitSetup");
  btn.disabled = true;
  btn.textContent = currentLang === "de" ? "⚙️ Initialisiere Mesh-VPN, DNS und Zertifikate..." : "⚙️ Initializing Mesh-VPN, DNS and certificates...";

  try {
    const res = await fetch("/api/setup", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        base_domain: baseDomain,
        admin_username: adminUser,
        admin_password: adminPass
      })
    });
    const data = await res.json();
    if (!res.ok) {
      alert("Fehler: " + (data.error || "Setup fehlgeschlagen"));
      btn.disabled = false;
      btn.textContent = i18n[currentLang].btn_start_setup;
      return;
    }

    sessionToken = data.session_token;
    localStorage.setItem("benzcloud_token", sessionToken);
    checkSystemStatus();
  } catch (err) {
    alert("Netzwerkfehler: " + err);
    btn.disabled = false;
  }
}

async function submitLogin(e) {
  e.preventDefault();
  const username = document.getElementById("loginUser").value.trim();
  const password = document.getElementById("loginPass").value;

  try {
    const res = await fetch("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password })
    });
    const data = await res.json();
    if (!res.ok) {
      alert(currentLang === "de" ? "Ungültige Anmeldedaten" : "Invalid credentials");
      return;
    }

    sessionToken = data.token;
    localStorage.setItem("benzcloud_token", sessionToken);
    checkSystemStatus();
  } catch (err) {
    alert("Netzwerkfehler: " + err);
  }
}

function logout() {
  sessionToken = "";
  localStorage.removeItem("benzcloud_token");
  location.reload();
}

function switchTab(tabId) {
  document.querySelectorAll(".tab-btn").forEach(btn => btn.classList.remove("active"));
  document.querySelectorAll(".tab-pane").forEach(pane => pane.classList.remove("active"));

  event.target.classList.add("active");
  const targetPane = document.getElementById(`tabContent_${tabId}`);
  if (targetPane) targetPane.classList.add("active");

  refreshActiveTabData();
}

function refreshActiveTabData() {
  if (!sessionToken || !systemConfig || !systemConfig.setup_completed) return;
  const activePane = document.querySelector(".tab-pane.active");
  if (!activePane) return;

  const id = activePane.id;
  if (id === "tabContent_drive") loadDriveFiles();
  if (id === "tabContent_vpn") loadVPNPeers();
  if (id === "tabContent_dns") loadDNSData();
  if (id === "tabContent_plugins") loadPlugins();
  if (id === "tabContent_users") loadUsers();
}

function loadDashboardData() {
  loadDriveFiles();
  loadVPNPeers();
  loadDNSData();
  loadPlugins();
  loadUsers();
}

/* DRIVE */
async function loadDriveFiles() {
  try {
    const res = await fetch(`/api/drive/files?path=${encodeURIComponent(currentPath)}`, {
      headers: { Authorization: `Bearer ${sessionToken}` }
    });
    if (!res.ok) return;
    const data = await res.json();
    const tbody = document.getElementById("fileTableBody");
    tbody.innerHTML = "";

    if (!data.files || data.files.length === 0) {
      tbody.innerHTML = `<tr><td colspan="4" class="text-center text-muted">${currentLang === "de" ? "Keine Dateien vorhanden." : "No files available."}</td></tr>`;
      return;
    }

    data.files.forEach(f => {
      const tr = document.createElement("tr");
      const icon = f.is_dir ? "📁" : "📄";
      const sizeStr = f.is_dir ? "-" : formatBytes(f.size);
      const dateStr = new Date(f.mod_time).toLocaleString();

      tr.innerHTML = `
        <td>${icon} <strong>${escapeHtml(f.name)}</strong></td>
        <td>${sizeStr}</td>
        <td>${dateStr}</td>
        <td>
          ${!f.is_dir ? `<a class="btn-sm btn-primary" href="/api/drive/download?path=${encodeURIComponent(f.path)}&token=${sessionToken}">${i18n[currentLang].btn_download}</a>` : ""}
          <button class="btn-sm btn-outline" onclick="deleteFile('${escapeHtml(f.path)}')">${i18n[currentLang].btn_delete}</button>
        </td>
      `;
      tbody.appendChild(tr);
    });
  } catch (err) {
    console.error("Drive load failed:", err);
  }
}

async function uploadSelectedFile() {
  const input = document.getElementById("fileUploadInput");
  if (!input.files || input.files.length === 0) return;
  const file = input.files[0];
  const targetPath = currentPath ? `${currentPath}/${file.name}` : file.name;

  try {
    const res = await fetch(`/api/drive/upload?path=${encodeURIComponent(targetPath)}`, {
      method: "POST",
      headers: { Authorization: `Bearer ${sessionToken}` },
      body: file
    });
    if (res.ok) {
      loadDriveFiles();
    } else {
      alert("Upload failed");
    }
  } catch (err) {
    alert("Upload error: " + err);
  }
  input.value = "";
}

async function promptNewFolder() {
  const name = prompt(i18n[currentLang].prompt_folder_name);
  if (!name) return;
  const folderPath = currentPath ? `${currentPath}/${name}` : name;
  try {
    await fetch("/api/drive/folder", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionToken}`
      },
      body: JSON.stringify({ path: folderPath })
    });
    loadDriveFiles();
  } catch (err) {
    alert("Folder creation failed");
  }
}

async function deleteFile(path) {
  if (!confirm(i18n[currentLang].confirm_delete)) return;
  try {
    await fetch(`/api/drive/delete?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${sessionToken}` }
    });
    loadDriveFiles();
  } catch (err) {
    alert("Delete failed");
  }
}

/* VPN PEERS */
async function loadVPNPeers() {
  try {
    const res = await fetch("/api/status");
    const data = await res.json();
    const tbody = document.getElementById("peerTableBody");
    tbody.innerHTML = "";

    // Server Node
    const sTr = document.createElement("tr");
    sTr.innerHTML = `
      <td><strong>${escapeHtml(data.system)} (Lighthouse)</strong></td>
      <td><code>${data.server_vpn_ip}</code></td>
      <td><span class="badge-tag badge-blue">Lighthouse</span></td>
      <td><span class="pulse-dot"></span> Online</td>
      <td>Permanenter Host</td>
    `;
    tbody.appendChild(sTr);
  } catch (err) {
    console.error("VPN peers load failed:", err);
  }
}

/* DNS & ROUTING */
async function loadDNSData() {
  if (!systemConfig) return;
  const domain = systemConfig.base_domain;
  const ip = systemConfig.server_vpn_ip;

  const recordsList = document.getElementById("dnsRecordsList");
  recordsList.innerHTML = `
    <div class="record-row"><span>vpn.${domain}</span><code>${ip}</code></div>
    <div class="record-row"><span>drive.${domain}</span><code>${ip}</code></div>
    <div class="record-row"><span>mail.${domain}</span><code>${ip}</code></div>
    <div class="record-row"><span>chat.${domain}</span><code>${ip}</code></div>
    <div class="record-row"><span>*.${domain} (Web Engine)</span><code>${ip}</code></div>
  `;

  try {
    const res = await fetch("/api/dns/logs", {
      headers: { Authorization: `Bearer ${sessionToken}` }
    });
    if (!res.ok) return;
    const data = await res.json();
    const logsList = document.getElementById("dnsLogsList");
    logsList.innerHTML = "";

    if (!data.logs || data.logs.length === 0) {
      logsList.innerHTML = `<div class="p-item text-muted">${currentLang === "de" ? "Noch keine externen DNS-Anfragen eingetroffen." : "No DNS queries logged yet."}</div>`;
      return;
    }

    data.logs.slice(-8).reverse().forEach(log => {
      const row = document.createElement("div");
      row.className = "log-row";
      row.innerHTML = `
        <span><strong>${escapeHtml(log.domain)}</strong> (${escapeHtml(log.type)})</span>
        <code>${escapeHtml(log.answer)}</code>
      `;
      logsList.appendChild(row);
    });
  } catch (err) {
    console.error("DNS logs load failed:", err);
  }
}

/* PLUGINS */
async function loadPlugins() {
  try {
    const res = await fetch("/api/plugins", {
      headers: { Authorization: `Bearer ${sessionToken}` }
    });
    if (!res.ok) return;
    const data = await res.json();
    const grid = document.getElementById("pluginCardsGrid");
    grid.innerHTML = "";

    data.plugins.forEach(p => {
      const card = document.createElement("div");
      card.className = "plugin-card";
      card.innerHTML = `
        <div class="p-header">
          <div class="p-title">${escapeHtml(p.name)}</div>
          <span class="badge-tag ${p.enabled ? 'badge-green' : 'badge-tag'}">${p.enabled ? i18n[currentLang].btn_toggle_on : i18n[currentLang].btn_toggle_off}</span>
        </div>
        <p class="text-muted" style="font-size:0.85rem;">Binary: <code>${escapeHtml(p.binary)}</code> | Port: <code>${p.port}</code></p>
        <p class="text-muted" style="font-size:0.85rem;">Subdomains: <code>${p.subdomains.length ? p.subdomains.join(', ') : 'Frei wählbar'}</code></p>
        <div style="margin-top:auto;">
          <button class="btn-sm ${p.enabled ? 'btn-outline' : 'btn-primary'}" onclick="togglePlugin('${p.id}', ${!p.enabled})">
            ${p.enabled ? (currentLang === 'de' ? 'Deaktivieren' : 'Disable') : (currentLang === 'de' ? 'Aktivieren' : 'Enable')}
          </button>
        </div>
      `;
      grid.appendChild(card);
    });
  } catch (err) {
    console.error("Plugins load failed:", err);
  }
}

async function togglePlugin(id, enabled) {
  try {
    await fetch("/api/plugins/toggle", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionToken}`
      },
      body: JSON.stringify({ id, enabled })
    });
    loadPlugins();
  } catch (err) {
    alert("Plugin toggle failed");
  }
}

/* USERS */
async function loadUsers() {
  try {
    const res = await fetch("/api/users", {
      headers: { Authorization: `Bearer ${sessionToken}` }
    });
    if (!res.ok) return;
    const data = await res.json();
    const tbody = document.getElementById("usersTableBody");
    tbody.innerHTML = "";

    data.users.forEach(u => {
      const tr = document.createElement("tr");
      let permsBadges = [];
      if (u.permissions.admin) permsBadges.push('<span class="badge-tag badge-blue">Admin</span>');
      if (u.permissions.drive) permsBadges.push('<span class="badge-tag badge-green">Drive</span>');
      if (u.permissions.mail) permsBadges.push('<span class="badge-tag badge-green">Mail</span>');
      if (u.permissions.chat) permsBadges.push('<span class="badge-tag badge-green">Chat</span>');
      if (u.permissions.web) permsBadges.push('<span class="badge-tag badge-green">Web</span>');

      tr.innerHTML = `
        <td><strong>${escapeHtml(u.username)}</strong></td>
        <td><code>${u.overlay_ip}</code></td>
        <td>${permsBadges.join(' ')}</td>
        <td><span class="badge-tag badge-locked">🔒 VPN & DNS (Immer aktiv)</span></td>
      `;
      tbody.appendChild(tr);
    });
  } catch (err) {
    console.error("Users load failed:", err);
  }
}

function openCreateUserModal() {
  document.getElementById("createUserModal").style.display = "flex";
}

function closeCreateUserModal() {
  document.getElementById("createUserModal").style.display = "none";
}

async function submitCreateUser(e) {
  e.preventDefault();
  const username = document.getElementById("newUsername").value.trim();
  const password = document.getElementById("newPassword").value;
  const drive = document.getElementById("permDrive").checked;
  const mail = document.getElementById("permMail").checked;
  const chat = document.getElementById("permChat").checked;
  const web = document.getElementById("permWeb").checked;

  try {
    const res = await fetch("/api/users", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionToken}`
      },
      body: JSON.stringify({
        username,
        password,
        permissions: { admin: false, drive, mail, chat, web, vpn: true, dns: true }
      })
    });
    if (!res.ok) {
      alert("Fehler beim Erstellen des Benutzers");
      return;
    }
    closeCreateUserModal();
    loadUsers();
  } catch (err) {
    alert("Netzwerkfehler: " + err);
  }
}

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function escapeHtml(str) {
  if (!str) return "";
  return String(str).replace(/[&<>"']/g, m => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  }[m]));
}
