//go:build windows

package engine

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// GoodbyeDPI 0.2.3rc3 (TR) + WinDivert 2.2 (x86_64). Paket seviyesinde calisir: Discord'un guncelleyicisi
// dahil hicbir surecin proxy ayarina bakmasi gerekmez. Yonetici izni gerekir (WinDivert surucusu).
// Lisanslar: licenses/third_party.
var (
	//go:embed gdpi_windows/goodbyedpi.exe
	gdpiExe []byte
	//go:embed gdpi_windows/WinDivert.dll
	gdpiDLL []byte
	//go:embed gdpi_windows/WinDivert64.sys
	gdpiSys []byte
)

// Yandex DNS 1253 portu: ISS'nin DNS zehirlemesini asar (dip/turkey_dnsredir*.cmd ile ayni).
var gdpiDNS = []string{"--dns-addr", "77.88.8.8", "--dns-port", "1253", "--dnsv6-addr", "2a02:6b8::feed:0ff", "--dnsv6-port", "1253"}

func gdpiArgs(mode ...string) []string { return append(append([]string{}, mode...), gdpiDNS...) }

// DefaultProfiles: dip/turkey_dnsredir*.cmd on ayarlari (siralama = deneme sirasi). Son ikisi DNS yonlendirmesiz.
var DefaultProfiles = []Profile{
	{Name: "gdpi-9+dns", Args: gdpiArgs("-9")},
	{Name: "gdpi-5+dns", Args: gdpiArgs("-5")},
	{Name: "gdpi-ttl3+dns", Args: gdpiArgs("--set-ttl", "3")},
	{Name: "gdpi-5-ttl5+dns", Args: gdpiArgs("-5", "--set-ttl", "5")},
	{Name: "gdpi-9", Args: []string{"-9"}},
	{Name: "gdpi-5", Args: []string{"-5"}},
}

// GoodbyeDPI: goodbyedpi.exe surecini yonetir. Engine arayuzunde adres "" doner (sistem geneli, proxy yok).
type GoodbyeDPI struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	done chan struct{}
}

func (g *GoodbyeDPI) SystemWide() bool { return true }

// IsElevated: surec yonetici yetkisiyle mi calisiyor.
func IsElevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.b.Len() < 4096 {
		s.b.Write(p)
	}
	return len(p), nil
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.b.String())
}

func writeIfDifferent(path string, data []byte) error {
	// #nosec G304 -- yol kullanici girdisi degil: UserCacheDir + sabit ad
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, data) {
		return nil
	}
	tmp := path + ".tmp"
	// #nosec G306 -- calistirilabilir; dizin kullaniciya ozel
	if err := os.WriteFile(tmp, data, 0o700); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// extractGDPI: gomulu dosyalari %LOCALAPPDATA%\maxdpi\gdpi altina cikarir (surucu dll ile ayni dizinde olmali).
func extractGDPI() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "maxdpi", "gdpi")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	files := []struct {
		name string
		data []byte
	}{{"goodbyedpi.exe", gdpiExe}, {"WinDivert.dll", gdpiDLL}, {"WinDivert64.sys", gdpiSys}}
	for _, f := range files {
		if err := writeIfDifferent(filepath.Join(dir, f.name), f.data); err != nil {
			return "", fmt.Errorf("%s yazilamadi (antivirus karantinasi olabilir): %w", f.name, err)
		}
	}
	return dir, nil
}

func runHidden(name string, args ...string) {
	// #nosec G204 -- sabit sistem programlari ve sabit argumanlar
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", name), args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	_ = cmd.Run()
}

// killStaleGDPI: onceki oturumdan veya elle acilmis goodbyedpi kalmissa temizler (cift WinDivert sorunu).
func killStaleGDPI() { runHidden("taskkill.exe", "/F", "/IM", "goodbyedpi.exe") }

// resetWinDivert: uyumsuz/takili WinDivert hizmetini kaldirir (yalnizca baslatma basarisiz olunca).
func resetWinDivert() {
	for _, svc := range []string{"WinDivert", "WinDivert14"} {
		runHidden("sc.exe", "stop", svc)
		runHidden("sc.exe", "delete", svc)
	}
}

func (g *GoodbyeDPI) Start(p Profile) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cmd != nil {
		return "", errors.New("motor zaten calisiyor")
	}
	if !IsElevated() {
		return "", errors.New("yonetici izni gerekli: uygulamayi 'Yonetici olarak calistir' ile acin")
	}
	dir, err := extractGDPI()
	if err != nil {
		return "", fmt.Errorf("motor hazirlanamadi: %w", err)
	}
	killStaleGDPI()

	out, err := g.startOnce(dir, p)
	if err != nil && strings.Contains(strings.ToLower(out), "windivert") {
		resetWinDivert() // baska surumden kalan surucu
		_, err = g.startOnce(dir, p)
	}
	return "", err
}

// startOnce: goodbyedpi'yi baslatir; ilk 2 sn icinde kapanirsa cikti ile birlikte hata doner.
func (g *GoodbyeDPI) startOnce(dir string, p Profile) (string, error) {
	// #nosec G204 -- path: kendi cikardigimiz binary; args: kod icindeki sabit profil
	cmd := exec.Command(filepath.Join(dir, "goodbyedpi.exe"), p.Args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	buf := &syncBuf{}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("goodbyedpi baslatilamadi: %w", err)
	}
	if err := attach(cmd); err != nil { // uygulama olse bile surec ve dolayisiyla paket filtresi kapansin
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", fmt.Errorf("goodbyedpi surec grubuna baglanamadi: %w", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
		out := buf.String()
		if out == "" {
			out = "cikti yok"
		}
		return out, fmt.Errorf("goodbyedpi hemen kapandi: %s", out)
	case <-time.After(2 * time.Second):
	}
	g.cmd, g.done = cmd, done
	return "", nil
}

func (g *GoodbyeDPI) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cmd == nil {
		return nil
	}
	err := g.cmd.Process.Kill()
	<-g.done
	g.cmd, g.done = nil, nil
	return err
}
