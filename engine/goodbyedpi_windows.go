//go:build windows

package engine

import (
	"bytes"
	"crypto/sha256"
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
	"unsafe"

	"golang.org/x/sys/windows"
)

// GoodbyeDPI 0.2.3rc3 (TR) + WinDivert 2.2 (x86_64). Paket seviyesinde calisir: Discord'un guncelleyicisi
// dahil hicbir surecin proxy ayarina bakmasi gerekmez. Yonetici izni gerekir (WinDivert surucusu).
// Kaynak ve SHA-256: THIRD_PARTY.md. Lisanslar: licenses/third_party.
var (
	//go:embed gdpi_windows/goodbyedpi.exe
	gdpiExe []byte
	//go:embed gdpi_windows/WinDivert.dll
	gdpiDLL []byte
	//go:embed gdpi_windows/WinDivert64.sys
	gdpiSys []byte
)

// Yandex DNS 1253 portu: ISS'nin DNS zehirlemesini asar (GoodbyeDPI-Turkey turkey_dnsredir*.cmd ile ayni).
// Dikkat: sistemin TUM DNS sorgulari (yalniz Discord degil) sifresiz UDP ile Yandex'e gider (README: Gizlilik).
var gdpiDNS = []string{"--dns-addr", "77.88.8.8", "--dns-port", "1253", "--dnsv6-addr", "2a02:6b8::feed:0ff", "--dnsv6-port", "1253"}

func gdpiArgs(mode ...string) []string { return append(append([]string{}, mode...), gdpiDNS...) }

// DefaultProfiles: GoodbyeDPI-Turkey turkey_dnsredir*.cmd on ayarlari (siralama = deneme sirasi). Son ikisi DNS yonlendirmesiz.
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
	// #nosec G304 -- yol kullanici girdisi degil: yonetici-yazilabilir sabit dizin + sabit ad
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, data) {
		return nil
	}
	tmp := path + ".tmp"
	// #nosec G306 -- calistirilabilir; dizin yalnizca Yoneticiler/SYSTEM tarafindan yazilabilir
	if err := os.WriteFile(tmp, data, 0o700); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// gdpiDirSDDL: yalnizca SYSTEM ve Yoneticiler tam yetkili, miras kapali. Yonetici yetkisiyle calisan ikililer
// kullanici-yazilabilir bir dizinden calistirilirsa standart kullanici sureci yetki yukseltebilir.
const gdpiDirSDDL = "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"

// secureGDPIDir: %ProgramData%\maxdpi-gdpi dizinini kisitli DACL ile atomik olusturur. Onceden varsa
// (baska biri onceden olusturmus olabilir) reparse point olmadigini ve sahibinin Yoneticiler/SYSTEM oldugunu dogrular.
func secureGDPIDir() (string, error) {
	base, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "maxdpi-gdpi")
	sd, err := windows.SecurityDescriptorFromString(gdpiDirSDDL)
	if err != nil {
		return "", err
	}
	sa := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return "", err
	}
	if err := windows.CreateDirectory(p, sa); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return "", err
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return "", err
	}
	if attrs&windows.FILE_ATTRIBUTE_DIRECTORY == 0 || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return "", fmt.Errorf("%s guvenli bir dizin degil", dir)
	}
	osd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return "", err
	}
	owner, _, err := osd.Owner()
	if err != nil || owner == nil {
		return "", fmt.Errorf("%s sahibi okunamadi", dir)
	}
	for _, t := range []windows.WELL_KNOWN_SID_TYPE{windows.WinBuiltinAdministratorsSid, windows.WinLocalSystemSid} {
		if sid, err := windows.CreateWellKnownSid(t); err == nil && owner.Equals(sid) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("%s sahibi Yoneticiler/SYSTEM degil: dizini silip yeniden deneyin", dir)
}

// extractGDPI: gomulu dosyalari yonetici-yazilabilir dizine cikarir (surucu dll ile ayni dizinde olmali) ve
// calistirmadan once SHA-256 ile gomulu bayt'lara karsi dogrular.
func extractGDPI() (string, error) {
	dir, err := secureGDPIDir()
	if err != nil {
		return "", err
	}
	files := []struct {
		name string
		data []byte
	}{{"goodbyedpi.exe", gdpiExe}, {"WinDivert.dll", gdpiDLL}, {"WinDivert64.sys", gdpiSys}}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := writeIfDifferent(path, f.data); err != nil {
			return "", fmt.Errorf("%s yazilamadi (antivirus karantinasi olabilir): %w", f.name, err)
		}
		// #nosec G304 -- ayni sabit yol
		got, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(got) != sha256.Sum256(f.data) {
			return "", fmt.Errorf("%s dogrulanamadi", f.name)
		}
	}
	return dir, nil
}

// systemDir: System32 yolu API'den alinir; SystemRoot ortam degiskeni kullanici tarafindan degistirilebilir.
func systemDir() string {
	d, err := windows.GetSystemDirectory()
	if err != nil {
		return `C:\Windows\System32`
	}
	return d
}

func runHidden(name string, args ...string) {
	// #nosec G204 -- sabit sistem programlari ve sabit argumanlar
	cmd := exec.Command(filepath.Join(systemDir(), name), args...)
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
