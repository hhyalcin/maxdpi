//go:build darwin

package engine

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// Tekrarli Baglan/Kes: surec, port ve goroutine sizintisi olmamali.
func TestLifecycleCycles(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		b := &ByeDPI{}
		addr, err := b.Start(DefaultProfiles[0])
		if err != nil {
			t.Fatalf("dongu %d: Start: %v", i, err)
		}
		pid := b.cmd.Process.Pid
		f, err := StartFrontend(addr, NewResolver())
		if err != nil {
			t.Fatalf("dongu %d: frontend: %v", i, err)
		}
		f.Close()
		if err := b.Stop(); err != nil {
			t.Fatalf("dongu %d: Stop: %v", i, err)
		}
		if alive(pid) {
			t.Fatalf("dongu %d: motor (pid %d) Stop sonrasi yasiyor", i, pid)
		}
		if err := b.Stop(); err != nil { // ikinci Stop zararsiz olmali
			t.Fatalf("dongu %d: ikinci Stop: %v", i, err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+3 {
		t.Errorf("goroutine sizintisi: %d -> %d", before, after)
	}
	if out, _ := exec.Command("/usr/bin/pgrep", "-f", "Caches/discord-unblock/ciadpi").Output(); len(out) > 0 {
		t.Errorf("yetim ciadpi kaldi: %s", out)
	}
}

// Cokme benzetimi: Stop cagrilmadan yeni oturum baslayinca eski yetim motor temizlenmeli.
func TestOrphanCleanupOnNextStart(t *testing.T) {
	old := &ByeDPI{}
	if _, err := old.Start(DefaultProfiles[0]); err != nil {
		t.Fatal(err)
	}
	oldPid := old.cmd.Process.Pid // Stop() CAGRILMIYOR: uygulama cokmus gibi

	fresh := &ByeDPI{}
	if _, err := fresh.Start(DefaultProfiles[0]); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fresh.Stop() }()
	_ = old.cmd.Wait() // killStale oldurdu; zombi birakma
	if alive(oldPid) {
		t.Errorf("yetim motor (pid %d) yeni oturumda temizlenmedi", oldPid)
	}
	if !alive(fresh.cmd.Process.Pid) {
		t.Error("yeni motor da oldurulmus")
	}
}

// Ayni ByeDPI ornegi iki kez Start edilemez.
func TestDoubleStartRejected(t *testing.T) {
	b := &ByeDPI{}
	if _, err := b.Start(DefaultProfiles[0]); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Stop() }()
	if _, err := b.Start(DefaultProfiles[0]); err == nil {
		t.Error("ikinci Start kabul edildi")
	}
}
