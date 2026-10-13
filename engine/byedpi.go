package engine

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Platform dosyalari saglar: ciadpiBin, ciadpiName, DefaultProfiles, prepareCmd, attach, killStale.

// ByeDPI: ciadpi surecini yonetir.
type ByeDPI struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

// extract: gomulu binary'yi kullaniciya ozel dizine cikarir; diskteki kopya gomulu ile ayni degilse yeniden yazar.
func extract() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "maxdpi")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// #nosec G302 -- dizin; 0700 (rwx yalnizca sahip) kasitli
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, ciadpiName)

	want := sha256.Sum256(ciadpiBin)
	// #nosec G304 -- yol kullanici girdisi degil: UserCacheDir + sabit ad
	if cur, err := os.ReadFile(path); err == nil && sha256.Sum256(cur) == want {
		return path, nil
	}
	// Yaz + yeniden adlandir: yaris/kismi yazma olmasin.
	tmp := path + ".tmp"
	// #nosec G306 -- calistirilabilir motor; dizin kullaniciya ozel, yalnizca sahip yazar
	if err := os.WriteFile(tmp, ciadpiBin, 0o700); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	// Yazdiktan sonra tekrar dogrula (baska surec degistirmis olabilir).
	// #nosec G304 -- ayni sabit yol
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, ciadpiBin) {
		return "", errors.New("motor dosyasi dogrulanamadi")
	}
	return path, nil
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func (b *ByeDPI) Start(p Profile) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil {
		return "", errors.New("motor zaten calisiyor")
	}
	path, err := extract()
	if err != nil {
		return "", fmt.Errorf("motor hazirlanamadi: %w", err)
	}
	killStale(path) // onceki oturum cokmus olabilir
	port, err := freePort()
	if err != nil {
		return "", err
	}
	// -i 127.0.0.1 zorunlu: ciadpi varsayilani 0.0.0.0 (yerel agdan erisilebilir).
	args := append([]string{"-i", "127.0.0.1", "-p", strconv.Itoa(port)}, p.Args...)
	// #nosec G204 -- path: dogrulanan gomulu binary; args: kod icindeki sabit profil + loopback/port
	cmd := exec.Command(path, args...)
	prepareCmd(cmd)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("motor baslatilamadi: %w", err)
	}
	if err := attach(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", fmt.Errorf("motor surec grubuna baglanamadi: %w", err)
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	// Port dinlemeye baslayana kadar bekle.
	for i := 0; i < 20; i++ {
		if c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err == nil {
			_ = c.Close()
			b.cmd = cmd
			return addr, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return "", errors.New("motor porta baglanmadi")
}

// Stop: ciadpi SIGTERM'i yok saydigi icin dogrudan sonlandirilir (macOS: SIGKILL, Windows: TerminateProcess).
func (b *ByeDPI) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil {
		return nil
	}
	err := b.cmd.Process.Kill()
	_ = b.cmd.Wait()
	b.cmd = nil
	return err
}
