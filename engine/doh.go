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
const dohEndpoint = "https://1.1.1.1/dns-query"

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
			Timeout: 6 * time.Second,
			Transport: &http.Transport{
				Proxy:                 nil, // sistem proxy'sine (kendimize) donmeyelim
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
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

	q := url.Values{"name": {host}, "type": {"A"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dohEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doh istegi: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh durum kodu: %d", resp.StatusCode)
	}

	var dr dohResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&dr); err != nil {
		return nil, fmt.Errorf("doh cevabi: %w", err)
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
