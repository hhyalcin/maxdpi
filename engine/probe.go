package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// ProbeDirect: sistem geneli motorda (GoodbyeDPI) discord.com ve Discord'un guncelleme sunucusuna
// sistem DNS'i + dogrudan TLS ile ulasilip ulasilmadigini dener (DNS yonlendirmesini de dogrular).
func ProbeDirect(ctx context.Context) error {
	for _, host := range []string{"discord.com", "updates.discord.com"} {
		if err := probeDirectHost(ctx, host); err != nil {
			return fmt.Errorf("%s: %w", host, err)
		}
	}
	return nil
}

func probeDirectHost(ctx context.Context, host string) error {
	tr := &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout:   6 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
		DisableKeepAlives:     true,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{
		Transport:     tr,
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("probe durum kodu: %d", resp.StatusCode)
	}
	return nil
}

// Probe: motorun SOCKS5 adresi uzerinden discord.com'a TLS ile ulasilip ulasilmadigini dener.
// Basari: sunucudan HTTP cevabi (5xx haric) gelmesi.
func Probe(ctx context.Context, socksAddr string, r *Resolver) error {
	ips, err := r.Lookup(ctx, "discord.com")
	if err != nil {
		return err
	}
	d, err := proxy.SOCKS5("tcp", socksAddr, nil, &net.Dialer{Timeout: 5 * time.Second})
	if err != nil {
		return err
	}
	cd := d.(proxy.ContextDialer)

	tr := &http.Transport{
		Proxy: nil,
		// URL host'u (discord.com) TLS SNI'da kalir; baglanti DoH ile bulunan IP'ye yapilir.
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return cd.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), "443"))
		},
		TLSHandshakeTimeout:   6 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
		DisableKeepAlives:     true,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{
		Transport:     tr,
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("probe durum kodu: %d", resp.StatusCode)
	}
	return nil
}
