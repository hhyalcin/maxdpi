//go:build windows

package engine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func updateExe() (string, error) {
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		return "", errors.New("LOCALAPPDATA yok")
	}
	p := filepath.Join(la, "Discord", "Update.exe")
	if _, err := os.Stat(p); err != nil {
		return "", errors.New("Discord kurulu degil")
	}
	return p, nil
}

// DiscordRunning: Discord zaten aciksa ortam degiskenleri ve proxy argumani uygulanamaz.
func DiscordRunning() bool {
	tl := filepath.Join(os.Getenv("SystemRoot"), "System32", "tasklist.exe")
	// #nosec G204 -- sabit sistem programi ve sabit argumanlar
	out, err := exec.Command(tl, "/FI", "IMAGENAME eq Discord.exe", "/NH", "/FO", "CSV").Output()
	return err == nil && strings.Contains(string(out), "Discord.exe")
}

// LaunchDiscord: Discord'u yerel onyuzu kullanacak sekilde baslatir (macOS ile ayni mantik).
//   - --proxy-server: Chromium icin (Squirrel'in --process-start-args'i ile iletilir).
//   - HTTP(S)_PROXY: guncelleyici yalnizca bunlari okur.
func LaunchDiscord(proxyAddr string) error {
	if DiscordRunning() {
		return errors.New("Discord acik: once tamamen kapatin")
	}
	update, err := updateExe()
	if err != nil {
		return err
	}
	u := "http://" + proxyAddr
	// #nosec G204 -- yol LOCALAPPDATA altindaki Update.exe; proxyAddr kendi loopback adresimiz
	cmd := exec.Command(update, "--processStart", "Discord.exe", "--process-start-args", "--proxy-server="+u)
	cmd.Env = append(os.Environ(), "HTTPS_PROXY="+u, "HTTP_PROXY="+u, "https_proxy="+u, "http_proxy="+u)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
