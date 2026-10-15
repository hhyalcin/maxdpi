# MaxDPI

Türkiye'de erişim engeli olan Discord'u tek tıkla açan masaüstü uygulaması (macOS + Windows).
Wails v2 + Go. Bu proje Discord Inc. ile bağlantılı, onaylı veya desteklenen bir proje değildir;
"Discord" ilgili sahibinin ticari markasıdır.

Kurulum ve işletim sistemi uyarıları için: [docs/KURULUM.md](docs/KURULUM.md).
Güvenlik açığı bildirimi için: [SECURITY.md](SECURITY.md).

## Nasıl çalışır

| | macOS | Windows |
|---|---|---|
| Motor | [ByeDPI](https://github.com/hufrea/byedpi) (`ciadpi`), kaynaktan derlenir | [GoodbyeDPI-Turkey](https://github.com/cagritaskn/GoodbyeDPI-Turkey) + WinDivert, gömülü |
| Yöntem | Yerel SOCKS5/HTTP CONNECT önyüzü (yalnızca `127.0.0.1`); Discord alan adları DoH ile çözülür, bağlantı ByeDPI'dan geçer | Paket seviyesinde, sistem geneli |
| Yetki | Yönetici gerekmez, sistem proxy'sine dokunulmaz | **Yönetici izni gerekir** (WinDivert çekirdek sürücüsü) |
| Doğrulama | Gerçek engelde giriş, mesaj, sesli kanal | Gerçek hatta Bağlan, güncelleyici, giriş, ses (2026-10-13) |

## Gizlilik ve güvenlik notları (okuyun)

- **Windows: DNS yönlendirmesi.** İlk dört profil, sistemin **tüm** DNS sorgularını (yalnızca Discord değil) şifresiz UDP ile
  Yandex DNS'e (`77.88.8.8:1253`, Rusya) yönlendirir. Sorgularınızı Yandex görür ve DNS cevapları şifrelenmemiştir.
  Bağlan, profilleri sırayla dener; son iki profil DNS yönlendirmesi yapmaz. Bağlantıyı kestiğinizde filtre durur.
- **Windows: çekirdek sürücüsü ve yönetici izni.** Paket yakalama için imzalı `WinDivert64.sys` yüklenir. Antivirüsler bunu
  işaretleyebilir. Uygulama imzasızdır (SmartScreen uyarısı beklenir). İkililer `engine/gdpi_windows/` altında depoda durur;
  kaynakları ve SHA-256 değerleri [THIRD_PARTY.md](THIRD_PARTY.md) içindedir ve `scripts/verify-gdpi.sh` ile doğrulanır.
  Motor dosyaları yalnızca Yöneticilerin yazabildiği `%ProgramData%\maxdpi-gdpi` altına çıkarılır.
- **macOS: DNS.** Discord alan adları DNS-over-HTTPS ile Cloudflare (1.1.1.1) ve yedek olarak Google (8.8.8.8) üzerinden çözülür.
- **Discord güncellemesi.** Windows'ta yalnızca yedek proxy yolunda, Discord açıkken `settings.json` içine geçici olarak
  `SKIP_HOST_UPDATE` yazılır (bu sürede Discord ana uygulama güncellemesi almaz); Discord kapanınca veya bağlantı kesilince geri alınır.
- **Discord'u yetkisiz başlatma.** Uygulama yönetici olarak çalışsa bile Discord'u Explorer üzerinden normal yetkiyle başlatır.
- Uygulama telemetri veya analitik göndermez. `MAXDPI_DEBUG=1` ile açılan günlük yalnızca
  Discord alan adlarını ve IP'lerini yerel dosyaya yazar.
- Bu tür araçların kullanımı ve dağıtımı ülkenizin mevzuatına tabi olabilir. Sorumluluk kullanıcıdadır; yazılım garanti
  verilmeden "olduğu gibi" sunulur (bkz. `LICENSE`).

## Derleme

Motor binary'leri (macOS: ByeDPI) depoda yoktur; sabit commit'ten kaynaktan üretilir (`go:embed` bunlara ihtiyaç duyar).
Windows'ta GoodbyeDPI ikilileri depodadır, ayrıca `engine/byedpi_windows.go` yedek `ciadpi.exe` ister.

```sh
scripts/build-engines.sh        # macOS: ciadpi, Windows: ciadpi.exe (mingw-w64 varsa)
scripts/verify-gdpi.sh          # gömülü GoodbyeDPI dosyalarını resmi paketle karşılaştırır
wails build                     # macOS
wails build -platform windows/amd64
```

Gereksinimler: Go, Node, Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0`),
Windows hedefi için `brew install mingw-w64`.

## Test

```sh
go vet ./... && GOOS=windows GOARCH=amd64 go vet ./...
go test ./engine
MAXDPI_E2E=1 go test ./engine -run E2E -v     # gerçek engelli hatta, ağ gerekir
govulncheck ./...
```

Teşhis günlüğü: `MAXDPI_DEBUG=1` ile `<önbellek dizini>/maxdpi/debug.log` (1 MiB'i aşınca sıfırlanır).

## Lisans

MaxDPI: [MIT](LICENSE). Üçüncü taraf bileşenler ve lisansları: [THIRD_PARTY.md](THIRD_PARTY.md).
