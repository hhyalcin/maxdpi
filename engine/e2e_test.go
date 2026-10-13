//go:build darwin || windows

package engine

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// Gercek engelli hatta calisir: MAXDPI_E2E=1 go test ./engine -run E2E -v
func TestE2EDiscordViaFrontend(t *testing.T) {
	if os.Getenv("MAXDPI_E2E") != "1" {
		t.Skip("MAXDPI_E2E=1 degil")
	}
	r := NewResolver()
	b := &ByeDPI{}
	defer b.Stop()

	var addr string
	for _, p := range DefaultProfiles {
		a, err := b.Start(p)
		if err != nil {
			t.Fatalf("%s baslamadi: %v", p.Name, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = Probe(ctx, a, r)
		cancel()
		t.Logf("profil %-16s probe: %v", p.Name, err)
		if err == nil {
			addr = a
			break
		}
		b.Stop()
	}
	if addr == "" {
		t.Fatal("hicbir profil calismadi")
	}

	f, err := StartFrontend(addr, r)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Iki protokol: Chromium/--proxy-server (socks5 veya http) ve Rust guncelleyici (HTTPS_PROXY = http CONNECT).
	// Alan adi proxy'ye verilir, cozumleme onyuzde DoH ile yapilir.
	for _, scheme := range []string{"socks5", "http"} {
		pu, _ := url.Parse(scheme + "://" + f.Addr())
		client := &http.Client{
			Timeout:   15 * time.Second,
			Transport: &http.Transport{Proxy: http.ProxyURL(pu)},
		}
		for _, u := range []string{"https://discord.com/", "https://updates.discord.com/", "https://gateway.discord.gg/", "https://cdn.discordapp.com/"} {
			resp, err := client.Get(u)
			if err != nil {
				t.Errorf("[%s] %s: %v", scheme, u, err)
				continue
			}
			_ = resp.Body.Close()
			t.Logf("[%s] %s -> %d", scheme, u, resp.StatusCode)
		}
	}
}
