// Package engine: DPI atlatma motorunu (ByeDPI) yonetir, DoH ile cozumleyen yerel SOCKS5 onyuzu saglar.
package engine

import "strings"

// Profile: motora verilen sabit arguman seti. Argumanlar kod icinde tanimlidir, kullanici girdisi degildir.
type Profile struct {
	Name string
	Args []string
}

// Engine: platforma ozel DPI motoru. Start, motorun dinledigi SOCKS5 adresini (127.0.0.1:port) dondurur.
type Engine interface {
	Start(p Profile) (socksAddr string, err error)
	Stop() error
}

// discordSuffixes: DoH + motor uzerinden gidecek alan adlari. Digerleri dogrudan baglanir.
var discordSuffixes = []string{
	"discord.com", "discord.gg", "discordapp.com", "discordapp.net",
	"discord.media", "discordcdn.com", "discordstatus.com", "discord.gift",
}

// cfHTTPSPorts: Cloudflare'in alternatif HTTPS portlari. Discord ses sinyalleri (c-*.discord.media) 2083 gibi portlari kullanir.
var cfHTTPSPorts = map[int]bool{2053: true, 2083: true, 2087: true, 2096: true, 8443: true}

// portAllowed: herkese 80/443; ek olarak yalnizca Discord alan adlarina Cloudflare HTTPS portlari.
func portAllowed(host string, isIP bool, port int) bool {
	if port == 80 || port == 443 {
		return true
	}
	return !isIP && isDiscordHost(host) && cfHTTPSPorts[port]
}

// isDiscordHost: tam eslesme veya "." sinirli alt alan adi eslesmesi.
func isDiscordHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range discordSuffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}
