//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

// LaunchGUI launches Windows app mode using Edge or Chrome without CGO.
func LaunchGUI(title, url string, width, height int) {
	appArg := fmt.Sprintf("--app=%s", url)
	windowSizeArg := fmt.Sprintf("--window-size=%d,%d", width, height)

	// Try Microsoft Edge
	cmd := exec.Command("msedge.exe", appArg, windowSizeArg)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	if err := cmd.Start(); err == nil {
		_ = cmd.Wait()
		return
	}

	// Try Chrome
	cmdChrome := exec.Command("chrome.exe", appArg, windowSizeArg)
	if err := cmdChrome.Start(); err == nil {
		_ = cmdChrome.Wait()
		return
	}

	// Fallback to default browser
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
