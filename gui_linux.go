//go:build linux && cgo

package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

static void activate_gtk_app(const char* title, const char* url, int width, int height) {
    gtk_init(NULL, NULL);

    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title);
    gtk_window_set_default_size(GTK_WINDOW(window), width, height);
    gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

    GtkWidget *web_view = webkit_web_view_new();
    gtk_container_add(GTK_CONTAINER(window), web_view);

    g_signal_connect(window, "destroy", G_CALLBACK(gtk_main_quit), NULL);

    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(web_view), url);
    gtk_widget_show_all(window);

    gtk_main();
}
*/
import "C"
import "unsafe"

// LaunchGUI launches native WebKitGTK desktop shell on Linux.
func LaunchGUI(title, url string, width, height int) {
	cTitle := C.CString(title)
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cURL))

	C.activate_gtk_app(cTitle, cURL, C.int(width), C.int(height))
}
