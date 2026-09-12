//go:build linux && cgo

package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <stdlib.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

static int check_display() {
    int argc = 0;
    char **argv = NULL;
    return gtk_init_check(&argc, &argv) ? 1 : 0;
}

static void on_window_destroy(GtkWidget *widget, gpointer data) {
    gtk_main_quit();
}

static gboolean on_context_menu(WebKitWebView *web_view, WebKitContextMenu *context_menu, GdkEvent *event, WebKitHitTestResult *hit_test_result, gpointer user_data) {
    return TRUE;
}

static void set_window_icon_from_memory(GtkWindow *window, const void *buf, gsize len) {
    if (!buf || len == 0) return;
    GError *err = NULL;
    GdkPixbufLoader *loader = gdk_pixbuf_loader_new();
    if (loader) {
        if (gdk_pixbuf_loader_write(loader, (const guint8 *)buf, len, &err)) {
            gdk_pixbuf_loader_close(loader, &err);
            GdkPixbuf *pixbuf = gdk_pixbuf_loader_get_pixbuf(loader);
            if (pixbuf) {
                gtk_window_set_icon(window, pixbuf);
                gtk_window_set_default_icon(pixbuf);
            }
        }
        g_object_unref(loader);
    }
    gtk_window_set_default_icon_name("benzcloud-server");
    gtk_window_set_icon_name(window, "benzcloud-server");
}

static void activate_gtk_app(const char* title, const char* url, int width, int height, const void *icon_buf, int icon_len) {
    int argc = 0;
    char **argv = NULL;
    if (!gtk_init_check(&argc, &argv)) {
        return;
    }

    g_set_prgname("benzcloud-server");
    g_set_application_name("BenzCloud Server");

    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title);
    gtk_window_set_default_size(GTK_WINDOW(window), width, height);
    gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

    if (icon_buf && icon_len > 0) {
        set_window_icon_from_memory(GTK_WINDOW(window), icon_buf, (gsize)icon_len);
    }

    GdkRGBA bg_color;
    gdk_rgba_parse(&bg_color, "#0b0f19");

    WebKitSettings *settings = webkit_settings_new();
    webkit_settings_set_enable_developer_extras(settings, FALSE);
    webkit_settings_set_hardware_acceleration_policy(settings, WEBKIT_HARDWARE_ACCELERATION_POLICY_ALWAYS);

    GtkWidget *web_view = webkit_web_view_new_with_settings(settings);
    webkit_web_view_set_background_color(WEBKIT_WEB_VIEW(web_view), &bg_color);
    g_signal_connect(web_view, "context-menu", G_CALLBACK(on_context_menu), NULL);

    gtk_container_add(GTK_CONTAINER(window), web_view);
    g_signal_connect(window, "destroy", G_CALLBACK(on_window_destroy), NULL);

    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(web_view), url);
    gtk_widget_show_all(window);

    gtk_main();
}
*/
import "C"
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"
)

func init() {
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	_ = os.Setenv("WEBKIT_FORCE_COMPOSITING_MODE", "1")
}

// installDesktopIntegration automatically installs icons and desktop file into user's XDG directories
func installDesktopIntegration() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "512x512", "apps")
	pixmapDir := filepath.Join(home, ".local", "share", "pixmaps")
	appDir := filepath.Join(home, ".local", "share", "applications")
	_ = os.MkdirAll(iconDir, 0755)
	_ = os.MkdirAll(pixmapDir, 0755)
	_ = os.MkdirAll(appDir, 0755)

	iconPng, _ := webFS.ReadFile("web/icon.png")
	if len(iconPng) > 0 {
		_ = os.WriteFile(filepath.Join(iconDir, "benzcloud-server.png"), iconPng, 0644)
		_ = os.WriteFile(filepath.Join(pixmapDir, "benzcloud-server.png"), iconPng, 0644)
	}
	iconSvg, _ := webFS.ReadFile("web/icon.svg")
	if len(iconSvg) > 0 {
		svgDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "scalable", "apps")
		_ = os.MkdirAll(svgDir, 0755)
		_ = os.WriteFile(filepath.Join(svgDir, "benzcloud-server.svg"), iconSvg, 0644)
	}

	desktopPath := filepath.Join(appDir, "benzcloud-server.desktop")
	execPath, _ := os.Executable()
	if execPath == "" {
		execPath = "benzcloud-server"
	}
	content := fmt.Sprintf(`[Desktop Entry]
Name=BenzCloud Server
Comment=Decentralized Mesh Cloud & Self-Hosting Core
Exec=%s
Icon=benzcloud-server
Terminal=false
Type=Application
Categories=Network;Server;Utility;
StartupWMClass=benzcloud-server
X-Wayland-AppID=benzcloud-server
`, execPath)
	_ = os.WriteFile(desktopPath, []byte(content), 0644)
}

// LaunchGUI launches native WebKitGTK desktop shell on Linux.
func LaunchGUI(title, url string, width, height int) {
	installDesktopIntegration()

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	if !hasDisplay || C.check_display() == 0 {
		log.Println("[GUI] Kein Display gefunden, öffne Standard-Browser...")
		_ = exec.Command("xdg-open", url).Start()
		return
	}

	cTitle := C.CString(title)
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cURL))

	iconBytes, _ := webFS.ReadFile("web/icon.png")
	var iconPtr unsafe.Pointer
	if len(iconBytes) > 0 {
		iconPtr = unsafe.Pointer(&iconBytes[0])
	}

	C.activate_gtk_app(cTitle, cURL, C.int(width), C.int(height), iconPtr, C.int(len(iconBytes)))
}
