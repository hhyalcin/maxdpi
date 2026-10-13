//go:build windows

package engine

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
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
//   - HTTP(S)_PROXY: updater.node (Rust/reqwest) bunlari okur.
//   - Discord'un gomulu guncelleyicisi (updater_client) ortam degiskenlerini, --proxy-server'i ve sistem
//     proxy'sini YOK SAYIP updates.discord.com'a dogrudan baglanir; engelli DNS'ten sahte sertifika alip
//     sonsuz "Update failed" dongusune girer (gercek Windows'ta dogrulandi). Bu yuzden Discord calistigi
//     surece settings.json'a SKIP_HOST_UPDATE yazilir, Discord kapaninca geri alinir. Sistem proxy'si
//     yalnizca MAXDPI_SYSTEM_PROXY=1 ile (varsayilan kapali; uygulama yalnizca Discord'u etkilemeli).
func LaunchDiscord(proxyAddr string) error {
	if DiscordRunning() {
		return errors.New("Discord acik: once tamamen kapatin")
	}
	update, err := updateExe()
	if err != nil {
		return err
	}
	if systemProxyEnabled {
		setSystemProxy(proxyAddr)
	}
	setSkipHostUpdate()
	watchDiscord()
	u := "http://" + proxyAddr
	// #nosec G204 -- yol LOCALAPPDATA altindaki Update.exe; proxyAddr kendi loopback adresimiz
	cmd := exec.Command(update, "--processStart", "Discord.exe", "--process-start-args", "--proxy-server="+u)
	cmd.Env = append(os.Environ(), "HTTPS_PROXY="+u, "HTTP_PROXY="+u, "https_proxy="+u, "http_proxy="+u)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// LaunchDiscordDirect: sistem geneli motor (GoodbyeDPI) calisirken Discord'u proxy'siz baslatir; guncelleyici dahil
// butun trafik zaten paket seviyesinde duzeltilir. Discord aciksa (ornegin guncelleme dongusunde) dokunmaz.
func LaunchDiscordDirect() error {
	if DiscordRunning() {
		return nil
	}
	update, err := updateExe()
	if err != nil {
		return err
	}
	// #nosec G204 -- yol LOCALAPPDATA altindaki Update.exe, sabit argumanlar
	cmd := exec.Command(update, "--processStart", "Discord.exe")
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// systemProxyEnabled: varsayilan kapali; MAXDPI_SYSTEM_PROXY=1 yalnizca-HTTPS sistem proxy'sini Discord omru boyunca acar.
var systemProxyEnabled = os.Getenv("MAXDPI_SYSTEM_PROXY") == "1"

const skipKey = "SKIP_HOST_UPDATE"

func discordSettingsPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "discord", "settings.json")
}

func skipMarkerPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "maxdpi", "skip_host_update.marker"), nil
}

// editDiscordSettings: settings.json'daki diger anahtarlari koruyarak tek anahtari ekler/siler.
func editDiscordSettings(set bool) bool {
	p := discordSettingsPath()
	m := map[string]json.RawMessage{}
	// #nosec G304 -- yol APPDATA + sabit ad
	if b, err := os.ReadFile(p); err == nil {
		if json.Unmarshal(b, &m) != nil {
			return false // bozuk dosyaya dokunma
		}
	} else if !os.IsNotExist(err) || !set {
		return false
	}
	_, has := m[skipKey]
	if set == has && (!set || string(m[skipKey]) == "true") {
		return false // zaten istenen durumda
	}
	if set {
		m[skipKey] = json.RawMessage("true")
	} else {
		delete(m, skipKey)
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil || os.MkdirAll(filepath.Dir(p), 0o700) != nil {
		return false
	}
	tmp := p + ".maxdpi.tmp"
	if os.WriteFile(tmp, out, 0o600) != nil {
		return false
	}
	if os.Rename(tmp, p) != nil {
		_ = os.Remove(tmp)
		return false
	}
	return true
}

// setSkipHostUpdate: yalnizca biz eklediysek isaret dosyasi birakir (cokmeden sonra RecoverSystemProxy geri alir).
func setSkipHostUpdate() {
	if !editDiscordSettings(true) {
		return
	}
	if mp, err := skipMarkerPath(); err == nil && os.MkdirAll(filepath.Dir(mp), 0o700) == nil {
		_ = os.WriteFile(mp, []byte("1"), 0o600)
	}
}

func clearSkipHostUpdate() {
	mp, err := skipMarkerPath()
	if err != nil {
		return
	}
	if _, err := os.Stat(mp); err != nil {
		return // biz eklemedik
	}
	editDiscordSettings(false)
	_ = os.Remove(mp)
}

var watching bool // proxyMu ile korunur

// watchDiscord: Discord surecleri bitince yaptigimiz gecici degisiklikleri geri alir (tek izleyici).
func watchDiscord() {
	proxyMu.Lock()
	if watching {
		proxyMu.Unlock()
		return
	}
	watching = true
	proxyMu.Unlock()
	go func() {
		defer func() {
			proxyMu.Lock()
			watching = false
			proxyMu.Unlock()
		}()
		// Discord'un baslamasini bekle (en fazla 60 sn), sonra kapanmasini.
		started := false
		for i := 0; i < 20 && !started; i++ {
			time.Sleep(3 * time.Second)
			started = DiscordRunning()
		}
		for started {
			time.Sleep(3 * time.Second)
			started = DiscordRunning()
		}
		RestoreSystemProxy()
		clearSkipHostUpdate()
	}()
}

// RestoreDiscordSettings: uygulama kapanirken (Disconnect) Discord kapaliysa settings.json'i geri alir.
func RestoreDiscordSettings() {
	if !DiscordRunning() {
		clearSkipHostUpdate()
	}
}

const internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type savedProxyState struct {
	Enable    uint32 `json:"enable"`
	Server    string `json:"server"`
	HasServer bool   `json:"hasServer"`
	Ours      string `json:"ours"` // yazdigimiz ProxyServer degeri; kullanici sonradan degistirdiyse dokunmayiz
}

var (
	proxyMu    sync.Mutex
	savedProxy *savedProxyState
)

func backupPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "maxdpi", "proxy_backup.json"), nil
}

// refreshWinINet: HKCU Internet Settings degisikligini zaten calisan/yeni surecler icin anlik gecerli kilar.
func refreshWinINet() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	setOpt := wininet.NewProc("InternetSetOptionW")
	const (
		internetOptionSettingsChanged = 39
		internetOptionRefresh         = 37
	)
	_, _, _ = setOpt.Call(0, internetOptionSettingsChanged, 0, 0)
	_, _, _ = setOpt.Call(0, internetOptionRefresh, 0, 0)
}

// setSystemProxy: yalnizca HTTPS icin HKCU WinINet proxy'sini onyuze yonlendirir (duz http dogrudan gider).
// Onceki degerler once diske yazilir; cokme olsa bile RecoverSystemProxy bir sonraki acilista geri yukler.
func setSystemProxy(proxyAddr string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer func() { _ = k.Close() }()

	ours := "https=" + proxyAddr
	proxyMu.Lock()
	if savedProxy == nil {
		enable, _, _ := k.GetIntegerValue("ProxyEnable")
		server, _, serr := k.GetStringValue("ProxyServer")
		st := &savedProxyState{Enable: uint32(enable), Server: server, HasServer: serr == nil, Ours: ours}
		if p, perr := backupPath(); perr == nil && os.MkdirAll(filepath.Dir(p), 0o700) == nil {
			if b, merr := json.Marshal(st); merr == nil {
				if os.WriteFile(p, b, 0o600) != nil {
					proxyMu.Unlock()
					return // yedek yazilamadiysa sistem ayarina dokunma
				}
			}
		}
		savedProxy = st
	} else {
		savedProxy.Ours = ours
	}
	proxyMu.Unlock()

	if k.SetStringValue("ProxyServer", ours) != nil {
		return
	}
	if k.SetDWordValue("ProxyEnable", 1) != nil {
		return
	}
	refreshWinINet()
}

func restoreFrom(saved *savedProxyState) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer func() { _ = k.Close() }()
	if cur, _, err := k.GetStringValue("ProxyServer"); err != nil || cur != saved.Ours {
		return // kullanici/baska program degistirmis; ezme
	}
	_ = k.SetDWordValue("ProxyEnable", saved.Enable)
	if saved.HasServer {
		_ = k.SetStringValue("ProxyServer", saved.Server)
	} else {
		_ = k.DeleteValue("ProxyServer")
	}
	refreshWinINet()
}

// RestoreSystemProxy: setSystemProxy'den onceki WinINet durumunu geri yukler (Disconnect/kapanista cagrilir).
func RestoreSystemProxy() {
	proxyMu.Lock()
	saved := savedProxy
	savedProxy = nil
	proxyMu.Unlock()
	if saved == nil {
		return
	}
	restoreFrom(saved)
	if p, err := backupPath(); err == nil {
		_ = os.Remove(p)
	}
}

// RecoverSystemProxy: onceki oturum cokmusse birakilan proxy ayarini geri yukler (uygulama acilisinda cagrilir).
func RecoverSystemProxy() {
	if !DiscordRunning() {
		clearSkipHostUpdate()
	}
	p, err := backupPath()
	if err != nil {
		return
	}
	// #nosec G304 -- yol UserConfigDir + sabit ad
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var saved savedProxyState
	if json.Unmarshal(b, &saved) == nil && saved.Ours != "" {
		restoreFrom(&saved)
	}
	_ = os.Remove(p)
}
