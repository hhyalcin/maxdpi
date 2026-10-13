package engine

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestIsDiscordHost(t *testing.T) {
	yes := []string{"discord.com", "gateway.discord.gg", "CDN.DiscordApp.com", "discord.com."}
	no := []string{"notdiscord.com", "discord.com.evil.io", "evildiscord.gg", "example.com", ""}
	for _, h := range yes {
		if !isDiscordHost(h) {
			t.Errorf("%q Discord sayilmali ama sayilmadi", h)
		}
	}
	for _, h := range no {
		if isDiscordHost(h) {
			t.Errorf("%q Discord sayilmamali ama sayildi", h)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.5", "192.168.1.1", "169.254.169.254", "0.0.0.0", "224.0.0.1"} {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s public sayilmamali", s)
		}
	}
	if !isPublicIP(net.ParseIP("162.159.138.232")) {
		t.Error("162.159.138.232 public sayilmali")
	}
}

// socksConnect: yerel onyuze ham SOCKS5 CONNECT gonderir, cevap kodunu dondurur.
func socksConnect(t *testing.T, addr string, req []byte) byte {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	c.Write([]byte{5, 1, 0})
	sel := make([]byte, 2)
	if _, err := io.ReadFull(c, sel); err != nil {
		t.Fatal(err)
	}
	c.Write(req)
	rep := make([]byte, 10)
	if _, err := io.ReadFull(c, rep); err != nil {
		t.Fatal(err)
	}
	return rep[1]
}

func TestFrontendRejects(t *testing.T) {
	// Upstream gercek bir motor olmak zorunda degil: reddedilen istekler dial'a ulasmaz.
	f, err := StartFrontend("127.0.0.1:1", NewResolver())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// 127.0.0.1:443 -> ozel adres
	if got := socksConnect(t, f.Addr(), []byte{5, 1, 0, 1, 127, 0, 0, 1, 0x01, 0xBB}); got != repFailure {
		t.Errorf("loopback IP reddedilmedi, kod=%d", got)
	}
	// 8.8.8.8:22 -> izin verilmeyen port
	if got := socksConnect(t, f.Addr(), []byte{5, 1, 0, 1, 8, 8, 8, 8, 0x00, 0x16}); got != repNotAllowed {
		t.Errorf("port 22 reddedilmedi, kod=%d", got)
	}
	// BIND komutu
	if got := socksConnect(t, f.Addr(), []byte{5, 2, 0, 1, 8, 8, 8, 8, 0x01, 0xBB}); got != repCmdNotSupp {
		t.Errorf("BIND reddedilmedi, kod=%d", got)
	}
}

// Acik (bosta) bir istemci baglantisi varken Close() takilmamali.
func TestFrontendCloseWithOpenConn(t *testing.T) {
	f, err := StartFrontend("127.0.0.1:1", NewResolver())
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.DialTimeout("tcp", f.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte{5, 1, 0}) // selamlama gonder, istegi gonderme
	time.Sleep(100 * time.Millisecond)

	done := make(chan struct{})
	go func() { f.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Close() acik baglantida takildi")
	}
}

func TestLookupRejectsEmpty(t *testing.T) {
	if _, err := NewResolver().Lookup(context.Background(), ""); err == nil {
		t.Error("bos host kabul edildi")
	}
}

// httpConnect: yerel onyuze ham HTTP CONNECT gonderir, durum satirini dondurur.
func httpConnect(t *testing.T, addr, req string) string {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	c.Write([]byte(req))
	buf := make([]byte, 64)
	n, _ := c.Read(buf)
	line, _, _ := strings.Cut(string(buf[:n]), "\r\n")
	return line
}

func TestFrontendHTTPConnectRejects(t *testing.T) {
	f, err := StartFrontend("127.0.0.1:1", NewResolver())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cases := []struct{ name, req, want string }{
		{"loopback IP", "CONNECT 127.0.0.1:443 HTTP/1.1\r\nHost: x\r\n\r\n", "HTTP/1.1 502 Bad Gateway"},
		{"izinsiz port", "CONNECT 8.8.8.8:22 HTTP/1.1\r\nHost: x\r\n\r\n", "HTTP/1.1 403 Forbidden"},
		{"GET reddi", "GET http://example.com/ HTTP/1.1\r\nHost: x\r\n\r\n", "HTTP/1.1 405 Method Not Allowed"},
		{"IPv6 literal", "CONNECT [::1]:443 HTTP/1.1\r\nHost: x\r\n\r\n", "HTTP/1.1 403 Forbidden"},
	}
	for _, c := range cases {
		if got := httpConnect(t, f.Addr(), c.req); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPortAllowed(t *testing.T) {
	cases := []struct {
		host string
		ip   bool
		port int
		want bool
	}{
		{"discord.com", false, 443, true},
		{"example.com", false, 80, true},
		{"c-fra18-8b9a8cb3.discord.media", false, 2083, true},
		{"gateway.discord.gg", false, 8443, true},
		{"example.com", false, 2083, false}, // Discord disi: sadece 80/443
		{"discord.com", false, 22, false},   // Discord icin de rastgele port yok
		{"162.159.1.1", true, 2083, false},  // IP literal: sadece 80/443
		{"discord.com.evil.io", false, 2083, false},
	}
	for _, c := range cases {
		if got := portAllowed(c.host, c.ip, c.port); got != c.want {
			t.Errorf("portAllowed(%q,%v,%d)=%v, want %v", c.host, c.ip, c.port, got, c.want)
		}
	}
}
