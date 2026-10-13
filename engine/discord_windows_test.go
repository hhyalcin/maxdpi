//go:build windows

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditDiscordSettings(t *testing.T) {
	d := t.TempDir()
	t.Setenv("APPDATA", d)
	p := filepath.Join(d, "discord", "settings.json")
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	_ = os.WriteFile(p, []byte(`{"a":1,"WINDOW_BOUNDS":{"x":5}}`), 0o600)
	if !editDiscordSettings(true) {
		t.Fatal("eklenmedi")
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"SKIP_HOST_UPDATE": true`) || !strings.Contains(string(b), `"WINDOW_BOUNDS"`) {
		t.Fatalf("yanlis: %s", b)
	}
	if editDiscordSettings(true) {
		t.Error("ikinci ekleme degisiklik yapmamali")
	}
	if !editDiscordSettings(false) {
		t.Fatal("silinmedi")
	}
	b, _ = os.ReadFile(p)
	if strings.Contains(string(b), "SKIP") || !strings.Contains(string(b), `"a"`) {
		t.Fatalf("yanlis: %s", b)
	}
}
