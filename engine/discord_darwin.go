//go:build darwin

package engine

import (
	"errors"
	"os/exec"
)

// DiscordRunning: Discord zaten aciksa ortam degiskenleri ve proxy argumani uygulanamaz.
func DiscordRunning() bool {
	return exec.Command("/usr/bin/pgrep", "-x", "Discord").Run() == nil
}

// LaunchDiscord: Discord'u yerel onyuzu kullanacak sekilde baslatir.
//   - --proxy-server: Chromium (arayuz, WebSocket, medya) icin.
//   - HTTP(S)_PROXY: Discord'un Rust guncelleyicisi (updater.node, reqwest) yalnizca bunlari okur;
//     sistem proxy'sine ve --proxy-server'a bakmaz.
func LaunchDiscord(proxyAddr string) error {
	if DiscordRunning() {
		return errors.New("Discord acik: once tamamen kapatin")
	}
	u := "http://" + proxyAddr
	// #nosec G204 -- sabit /usr/bin/open; proxyAddr kendi loopback adresimiz
	return exec.Command("/usr/bin/open", "-a", "Discord",
		"--env", "HTTPS_PROXY="+u, "--env", "HTTP_PROXY="+u,
		"--env", "https_proxy="+u, "--env", "http_proxy="+u,
		"--args", "--proxy-server="+u).Run()
}
