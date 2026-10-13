# Discord Unblock

Türkiye'de erişim engeli olan Discord'u tek tıkla açan masaüstü uygulaması (macOS + Windows).

## Nasıl çalışır
Yerel bir önyüz (SOCKS5 + HTTP CONNECT, yalnızca 127.0.0.1) Discord alan adlarını DNS-over-HTTPS ile çözer ve
bağlantıyı [ByeDPI](https://github.com/hufrea/byedpi) (`ciadpi`) üzerinden geçirir. Discord, bu önyüz
kullanılacak şekilde başlatılır (`--proxy-server` + `HTTP(S)_PROXY`; güncelleyici yalnızca ortam değişkenlerini okur).
Sistem proxy ayarlarına dokunulmaz.

## Derleme
Motor binary'leri depoda yoktur; sabit commit'ten kaynaktan üretilir (`go:embed` bunlara ihtiyaç duyar):

```sh
scripts/build-engines.sh        # macOS: ciadpi, Windows: ciadpi.exe (mingw-w64 varsa)
wails build                     # macOS
wails build -platform windows/amd64
```

Gereksinimler: Go, Node, Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0`),
Windows motoru için `brew install mingw-w64`.

## Test
```sh
go test ./engine
DU_E2E=1 go test ./engine -run E2E -v     # gerçek engelli hatta, ağ gerekir
```
Teşhis günlüğü: `DU_DEBUG=1` ile `<önbellek dizini>/discord-unblock/debug.log`.

## Durum
macOS: gerçek engelde giriş, mesaj ve sesli kanal doğrulandı. Windows: derleniyor, henüz çalıştırılıp doğrulanmadı.
Lisans notları: ByeDPI MIT.
