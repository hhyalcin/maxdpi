package engine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Frontend: Discord'un baglandigi yerel SOCKS5 sunucusu.
// Discord alan adlarini DoH ile cozer, IP olarak motora (ByeDPI SOCKS5) iletir; digerleri dogrudan gider.
type Frontend struct {
	ln       net.Listener
	resolver *Resolver
	upstream proxy.ContextDialer
	wg       sync.WaitGroup

	mu     sync.Mutex
	conns  map[net.Conn]struct{} // Close() aktif baglantilari kapatabilsin
	closed bool
}

// track: baglantiyi kaydeder; Frontend kapaliysa false doner (cagiran baglantiyi kapatmali).
func (f *Frontend) track(c net.Conn) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return false
	}
	f.conns[c] = struct{}{}
	return true
}

func (f *Frontend) untrack(c net.Conn) {
	f.mu.Lock()
	delete(f.conns, c)
	f.mu.Unlock()
}

// StartFrontend: yalnizca 127.0.0.1 uzerinde rastgele portta dinler.
func StartFrontend(upstreamAddr string, r *Resolver) (*Frontend, error) {
	d, err := proxy.SOCKS5("tcp", upstreamAddr, nil, &net.Dialer{Timeout: 8 * time.Second})
	if err != nil {
		return nil, err
	}
	cd, ok := d.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("upstream dialer context desteklemiyor")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	f := &Frontend{ln: ln, resolver: r, upstream: cd, conns: make(map[net.Conn]struct{})}
	f.wg.Add(1)
	go f.serve()
	return f, nil
}

func (f *Frontend) Addr() string { return f.ln.Addr().String() }

// Close: dinlemeyi durdurur, aktif tum baglantilari kapatir ve goroutine'lerin bitmesini bekler.
func (f *Frontend) Close() {
	_ = f.ln.Close()
	f.mu.Lock()
	f.closed = true
	for c := range f.conns {
		_ = c.Close()
	}
	f.mu.Unlock()
	f.wg.Wait()
}

func (f *Frontend) serve() {
	defer f.wg.Done()
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			f.handle(c)
		}()
	}
}

// debug: DU_DEBUG=1 iken Discord'a yonlenen baglantilar <onbellek dizini>/discord-unblock/debug.log'a yazilir
// (yalnizca Discord alan adlari ve IP'ler). Windows'ta GUI uygulamasinin stderr'i gorunmedigi icin dosyaya yazilir.
var debug = os.Getenv("DU_DEBUG") == "1"

func init() {
	if !debug {
		return
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return
	}
	dir := filepath.Join(base, "discord-unblock")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	// #nosec G304 -- yol UserCacheDir + sabit ad
	if f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		log.SetOutput(f)
	}
}

func dbg(format string, args ...any) {
	if debug {
		log.Printf(format, args...)
	}
}

// SOCKS5 cevap kodlari (RFC 1928)
const (
	repOK          = 0x00
	repFailure     = 0x01
	repNotAllowed  = 0x02
	repCmdNotSupp  = 0x07
	repAtypNotSupp = 0x08
)

// target: istemcinin baglanmak istedigi hedef.
type target struct {
	host string
	isIP bool
	port int
}

// verdict: istemciye protokolune uygun cevap verir (ok / izin yok / hata).
type verdict struct {
	ok, deny, fail func()
}

func socksReply(c net.Conn, code byte) {
	_, _ = c.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
}

// readSOCKS: SOCKS5 el sikismasini ve CONNECT istegini okur. Gecersizse (ve gerekirse cevap verip) false doner.
func readSOCKS(c net.Conn, br *bufio.Reader) (target, verdict, bool) {
	var t target
	v := verdict{
		ok:   func() { socksReply(c, repOK) },
		deny: func() { socksReply(c, repNotAllowed) },
		fail: func() { socksReply(c, repFailure) },
	}
	// Selamlama: VER NMETHODS METHODS...
	h := make([]byte, 2)
	if _, err := io.ReadFull(br, h); err != nil || h[0] != 5 {
		return t, v, false
	}
	if _, err := io.CopyN(io.Discard, br, int64(h[1])); err != nil {
		return t, v, false
	}
	if _, err := c.Write([]byte{5, 0}); err != nil { // kimlik dogrulamasiz (yalnizca loopback)
		return t, v, false
	}
	// Istek: VER CMD RSV ATYP ADDR PORT
	rq := make([]byte, 4)
	if _, err := io.ReadFull(br, rq); err != nil || rq[0] != 5 {
		return t, v, false
	}
	if rq[1] != 1 { // sadece CONNECT
		socksReply(c, repCmdNotSupp)
		return t, v, false
	}
	switch rq[3] {
	case 1: // IPv4
		b := make([]byte, 4)
		if _, err := io.ReadFull(br, b); err != nil {
			return t, v, false
		}
		t.host, t.isIP = net.IP(b).String(), true
	case 3: // alan adi
		l := make([]byte, 1)
		if _, err := io.ReadFull(br, l); err != nil || l[0] == 0 {
			return t, v, false
		}
		b := make([]byte, l[0])
		if _, err := io.ReadFull(br, b); err != nil {
			return t, v, false
		}
		t.host = string(b)
	default: // IPv6 vb. desteklenmiyor
		socksReply(c, repAtypNotSupp)
		return t, v, false
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(br, pb); err != nil {
		return t, v, false
	}
	t.port = int(pb[0])<<8 | int(pb[1])
	return t, v, true
}

// readConnect: "CONNECT host:port HTTP/1.x" istegini okur (Discord'un Rust guncelleyicisi HTTPS_PROXY ile bunu kullanir).
func readConnect(c net.Conn, br *bufio.Reader) (target, verdict, bool) {
	var t target
	write := func(s string) func() { return func() { _, _ = c.Write([]byte(s)) } }
	v := verdict{
		ok:   write("HTTP/1.1 200 Connection established\r\n\r\n"),
		deny: write("HTTP/1.1 403 Forbidden\r\n\r\n"),
		fail: write("HTTP/1.1 502 Bad Gateway\r\n\r\n"),
	}
	// ReadSlice, tamponu (4096) asan satirda ErrBufferFull doner: asiri uzun satirlar reddedilir.
	line, err := br.ReadSlice('\n')
	if err != nil {
		return t, v, false
	}
	f := strings.Fields(string(line))
	if len(f) != 3 || !strings.HasPrefix(f[2], "HTTP/1.") {
		return t, v, false
	}
	if f[0] != "CONNECT" {
		write("HTTP/1.1 405 Method Not Allowed\r\n\r\n")()
		return t, v, false
	}
	// Basliklari bos satira kadar tuket (en fazla 32 satir).
	for i := 0; ; i++ {
		if i > 32 {
			return t, v, false
		}
		l, err := br.ReadSlice('\n')
		if err != nil {
			return t, v, false
		}
		if len(bytes.TrimSpace(l)) == 0 {
			break
		}
	}
	host, ps, err := net.SplitHostPort(f[1])
	if err != nil || host == "" {
		return t, v, false
	}
	port, err := strconv.Atoi(ps)
	if err != nil {
		return t, v, false
	}
	t.host, t.port = host, port
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil { // IPv6 literal desteklenmiyor
			write("HTTP/1.1 403 Forbidden\r\n\r\n")()
			return t, v, false
		}
		t.host, t.isIP = ip.To4().String(), true
	}
	return t, v, true
}

func (f *Frontend) handle(c net.Conn) {
	defer func() { _ = c.Close() }()
	if !f.track(c) {
		return
	}
	defer f.untrack(c)
	_ = c.SetDeadline(time.Now().Add(15 * time.Second)) // el sikisma suresi siniri

	br := bufio.NewReaderSize(c, 4096)
	first, err := br.Peek(1)
	if err != nil {
		return
	}
	var (
		t  target
		v  verdict
		ok bool
	)
	if first[0] == 5 {
		t, v, ok = readSOCKS(c, br)
	} else {
		t, v, ok = readConnect(c, br)
	}
	if !ok {
		return
	}
	routed := t.isIP || isDiscordHost(t.host) // diger alan adlari gunluge yazilmaz
	if !portAllowed(t.host, t.isIP, t.port) {
		if routed {
			dbg("RED   %s:%d (port izinli degil)", t.host, t.port)
		}
		v.deny()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	up, err := f.dial(ctx, t.host, t.isIP, t.port)
	if err != nil {
		if routed {
			dbg("HATA  %s:%d: %v", t.host, t.port, err)
		}
		v.fail()
		return
	}
	if routed {
		dbg("OK    %s:%d", t.host, t.port)
	}
	defer func() { _ = up.Close() }()
	if !f.track(up) {
		return
	}
	defer f.untrack(up)

	v.ok()
	_ = c.SetDeadline(time.Time{})
	relay(c, br, up)
}

// dial: hedefe yonlendirme karari.
//   - IP: loopback/ozel ise reddedilir, degilse motor uzerinden.
//   - Discord alan adi: DoH ile cozulur, motor uzerinden.
//   - Diger alan adlari: dogrudan.
func (f *Frontend) dial(ctx context.Context, host string, isIP bool, port int) (net.Conn, error) {
	ps := strconv.Itoa(port)
	if isIP {
		if !isPublicIP(net.ParseIP(host)) {
			return nil, errors.New("ozel adres reddedildi")
		}
		return f.upstream.DialContext(ctx, "tcp", net.JoinHostPort(host, ps))
	}
	if !isDiscordHost(host) {
		return (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(host, ps))
	}
	ips, err := f.resolver.Lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	var last error
	for i, ip := range ips {
		if i == 3 { // en fazla 3 IP dene
			break
		}
		conn, err := f.upstream.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), ps))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

// relay: iki yon kopyalar; biri bitince ikisini de kapatir. aR, a'nin tamponlu okuyucusudur.
func relay(a net.Conn, aR io.Reader, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, aR)
	<-done
	_ = a.Close()
	_ = b.Close()
	<-done
}
