package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// IP ile erisilir: DoH sunucusunu bulmak icin sistem DNS'ine (engelli) ihtiyac duyulmaz.
// Sirayla denenir (Cloudflare, sonra Google); hepsi JSON API sunar.
var dohEndpoints = []string{
	"https://1.1.1.1/dns-query",
	"https://1.0.0.1/dns-query",
	"https://8.8.8.8/resolve",
	"https://8.8.4.4/resolve",
}

type cacheEntry struct {
	ips []net.IP
	exp time.Time
}

// Resolver: Cloudflare DoH (JSON API) uzerinden A kaydi cozer, kisa sureli onbellekler.
type Resolver struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]cacheEntry
}

func NewResolver() *Resolver {
	return &Resolver{
		client: &http.Client{
			Timeout: 3 * time.Second, // sunucu basina; yedeklere gecmek icin kisa
			Transport: &http.Transport{
				Proxy:                 nil, // sistem proxy'sine (kendimize) donmeyelim
				TLSHandshakeTimeout:   2 * time.Second,
				ResponseHeaderTimeout: 2 * time.Second,
			},
		},
		cache: make(map[string]cacheEntry),
	}
}

type dohResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// isPublicIP: loopback/ozel/link-local/multicast adreslerini reddeder (DNS rebinding ile yerel servis erisimi onlenir).
func isPublicIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsUnspecified() || ip.IsMulticast())
}

// query: tek DoH sunucusuna sorar; ISS bir sunucuyu keserse Lookup siradakine gecer.
func (r *Resolver) query(ctx context.Context, endpoint, host string, dr *dohResponse) error {
	q := url.Values{"name": {host}, "type": {"A"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("doh istegi: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("doh durum kodu: %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(dr); err != nil {
		return fmt.Errorf("doh cevabi: %w", err)
	}
	return nil
}

func (r *Resolver) Lookup(ctx context.Context, host string) ([]net.IP, error) {
	if host == "" || len(host) > 253 {
		return nil, errors.New("gecersiz host")
	}

	r.mu.Lock()
	if e, ok := r.cache[host]; ok && time.Now().Before(e.exp) {
		r.mu.Unlock()
		return e.ips, nil
	}
	r.mu.Unlock()

	var dr dohResponse
	var lastErr error
	for _, ep := range dohEndpoints {
		if ctx.Err() != nil {
			break
		}
		if lastErr = r.query(ctx, ep, host, &dr); lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}

	var ips []net.IP
	ttl := 300
	for _, a := range dr.Answer {
		if a.Type != 1 { // sadece A; CNAME zinciri sunucuda acilmis gelir
			continue
		}
		ip := net.ParseIP(a.Data)
		if ip == nil || ip.To4() == nil || !isPublicIP(ip) {
			continue
		}
		ips = append(ips, ip)
		if a.TTL < ttl {
			ttl = a.TTL
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%s icin kullanilabilir A kaydi yok", host)
	}
	if ttl < 30 {
		ttl = 30
	}

	r.mu.Lock()
	r.cache[host] = cacheEntry{ips: ips, exp: time.Now().Add(time.Duration(ttl) * time.Second)}
	r.mu.Unlock()
	return ips, nil
}
