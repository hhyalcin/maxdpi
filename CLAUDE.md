# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Proje

MaxDPI: Türkiye'de erişim engelli Discord'u tek tıkla açan masaüstü uygulaması (Wails v2 + Go, ByeDPI gömülü). macOS'ta gerçek engelde giriş, mesaj ve sesli kanal doğrulandı. **Windows'ta farklı motor kullanılır:** gömülü GoodbyeDPI + WinDivert (paket seviyesi, sistem geneli, yönetici izni gerekir); gerçek hatta Bağlan, Discord güncelleyicisi, giriş ve ses doğrulandı (2026-10-13). Dizin adı (`discord-unblock`) eski; modül/uygulama adı `maxdpi`, bundle kimliği `com.maxdpi.app`.

## Komutlar

```sh
scripts/build-engines.sh            # ZORUNLU ilk adım: engine/bin_darwin/ciadpi (evrensel) + engine/bin_windows/ciadpi.exe (mingw-w64 varsa)
go vet ./... && GOOS=windows GOARCH=amd64 go vet ./...   # iki platformu da tip-kontrol et
go test ./engine                      # birim + yaşam döngüsü testleri
go test ./engine -run TestPortAllowed -v                 # tek test
MAXDPI_E2E=1 go test ./engine -run E2E -v                # GERÇEK engelli hat gerekir (ağ)
MAXDPI_E2E=1 go test . -run E2E -v                       # Connect + profil kaydı e2e (izole HOME)
govulncheck ./...                     # Go sürümü/bağımlılık güncellemelerinden sonra çalıştır
wails build                           # macOS (build/bin/maxdpi.app)
wails build -platform darwin/universal && scripts/make-dmg.sh   # dağıtım .dmg
wails build -platform windows/amd64   # build/bin/maxdpi.exe (macOS'tan çapraz)
```

- `engine/bin_*` depoda yok (gitignore); `go:embed` bunlar olmadan derlenmez. Kök paket de `frontend/dist` ister (`wails build` veya `cd frontend && npm run build` üretir).
- Windows motoru için `brew install mingw-w64`. Go toolchain `go.mod`'da sabit (`toolchain go1.27.2`).
- Teşhis günlüğü: `MAXDPI_DEBUG=1` ile `<UserCacheDir>/maxdpi/debug.log` (Discord'a yönlenen bağlantılar için `OK`/`RED`/`HATA`).

## Mimari

**Windows (GoodbyeDPI, sistem geneli):** `engine/goodbyedpi_windows.go` `GoodbyeDPI` motoru `engine/gdpi_windows/` altındaki gömülü `goodbyedpi.exe` + `WinDivert.dll` + `WinDivert64.sys` dosyalarını yalnız Yöneticiler/SYSTEM'in yazabildiği `%ProgramData%\maxdpi-gdpi\` altına (açık DACL, sahip doğrulaması, SHA-256 kontrolü) çıkarıp çalıştırır; `Discord`'u yetkisiz başlatmak için `unelevated` (Explorer ebeveyn) (github.com/cagritaskn/GoodbyeDPI-Turkey release-0.2.3rc3-turkey, `x86_64/`; kaynak ve SHA-256 `THIRD_PARTY.md`, doğrulama `scripts/verify-gdpi.sh`; lisanslar `licenses/third_party/`). `Start` boş adres döner (`engine.IsSystemWide`): yerel önyüz/DoH yok, `App.Connect` `engine.ProbeDirect` ile `discord.com` ve `updates.discord.com`'u sistem DNS'i + doğrudan TLS ile sınar, `OpenDiscord` `LaunchDiscordDirect` ile Discord'u proxy'siz açar. Profiller (`DefaultProfiles`): `-9`, `-5`, `--set-ttl 3`, `-5 --set-ttl 5` (hepsi Yandex DNS 77.88.8.8:1253 yönlendirmeli) + yönlendirmesiz `-9`/`-5`. Manifest `requireAdministrator` ister (`build/windows/wails.exe.manifest`); `Start` yönetici değilse hata verir. Neden proxy değil: Discord'un gömülü güncelleyicisi (`updater_client`) `--proxy-server`, `HTTP(S)_PROXY` ve WinINet proxy'sini yok sayıp doğrudan `updates.discord.com`'a bağlanır, zehirli DNS yüzünden "güvenilmeyen kök sertifika" ile sonsuz Update failed döngüsüne girer (SKIP_HOST_UPDATE de bu sürümde (93145) engellemez). Ciadpi proxy kodu (`byedpi*.go`, `socks.go`, `doh.go`) macOS için duruyor; Windows'ta kullanılmaz. Başka WinDivert/GoodbyeDPI çalışırken çakışır; `Start` `goodbyedpi.exe` süreçlerini öldürür, takılı `windivert` hizmetini başarısızlıkta siler. Antivirüs `WinDivert64.sys`'i karantinaya alabilir; Kaspersky ile çalışmaz.

**macOS (ByeDPI proxy):**

Discord'un TCP trafiği yerel bir önyüzden geçer; önyüz DNS'i kendisi (DoH) çözer ve bağlantıyı ByeDPI'ye IP olarak verir. Sistem proxy'sine dokunulmaz.

`App.Connect` (`app.go`): DoH ön kontrolü (profilden bağımsız, 8 sn) → profilleri sırala (`engine.OrderProfiles`, kayıtlı profil önde) → her biri için `ByeDPI.Start` + `engine.Probe` → ilk çalışanla `engine.StartFrontend` → profil adını kaydet. `OpenDiscord` Discord'u önyüze yönlendirerek başlatır.

- `engine/socks.go` `Frontend`: tek portta SOCKS5 **ve** HTTP CONNECT (ilk bayta göre ayrılır). Yönlendirme: Discord alan adı → DoH ile çöz → ByeDPI; IP literal → ByeDPI (özel/loopback IP reddedilir); diğer alan adlar → doğrudan. Port: herkese 80/443, yalnız Discord alan adlarına ek olarak Cloudflare HTTPS portları (`portAllowed`).
- `engine/byedpi.go` (ortak yaşam döngüsü) + platform dosyaları aynı sözleşmeyi sağlar: `ciadpiBin`, `ciadpiName`, `DefaultProfiles`, `prepareCmd`, `attach`, `killStale`. `byedpi_darwin.go` / `byedpi_windows.go`, `discord_darwin.go` / `discord_windows.go`, kökte `engine_darwin.go` / `engine_windows.go`. Yeni platform davranışı için önce bu sözleşmeye bak.
- Motor binary'si gömülüdür, kullanıcıya özel önbellek dizinine çıkarılır ve çalıştırmadan önce gömülü bayt'larla karşılaştırılır. Profil argümanları kod içindeki sabitlerdir; `config.json` yalnızca profil **adını** saklar.
- Arayüz (`frontend/`, vanilla TS) Go metodlarını `wailsjs/` bağlamalarıyla çağırır (`wails build` yeniden üretir). Harici linkler `App.OpenLink(id)` ile sabit tablodan açılır, arayüz keyfi URL gönderemez.

## Kolay kaçırılan tuzaklar

- **Discord güncelleyicisi (`updater.node`, Rust/reqwest) yalnızca `HTTP(S)_PROXY` okur.** `--proxy-server` ve sistem PAC'ı onu etkilemez; bu yüzden `LaunchDiscord` ikisini birden verir. Ortam değişkenleri ve argüman yalnızca yeni başlayan sürece uygulanır; Discord zaten açıksa `open` bunları iletmez, bu yüzden `LaunchDiscord` açık Discord'u reddeder.
- **(Yalnız Windows proxy yolu; artık varsayılan değil, bkz. yukarıdaki GoodbyeDPI notu)** `updater_client` yalnızca WinINet sistem proxy'sini kullanır** (env/`--proxy-server` yok sayılır; yoksa "güvenilmeyen kök sertifika" ile sonsuz Update failed — gerçek hatta doğrulandı). Bu yüzden Discord çalıştığı sürece `settings.json`'a `SKIP_HOST_UPDATE` yazılır (varsayılan, `watchDiscord` kapanınca geri alır). Yalnızca-HTTPS (`https=127.0.0.1:port`) WinINet proxy'si varsayılan **kapalı**dır (tüm uygulamaları etkiler); `MAXDPI_SYSTEM_PROXY=1` açar. DoH `engine/doh.go`'da 4 sunucu arasında yedekli (Cloudflare, Google). Önceki değer `config/maxdpi/proxy_backup.json`'a yazılır, çökme sonrası açılışta `RecoverSystemProxy` geri yükler.
- **Ses sinyali Cloudflare alt portlarını kullanır** (`c-*.discord.media`: 2083/2087/8443...). Port kısıtı gevşetilirse ses "bağlanıyor/kesildi" döngüsüne girer. Ses UDP'si proxy'den geçmez ve bu hatta engelli değildir; UDP'yi suçlamadan önce `debug.log`'a bak.
- **`killStale` aynı yoldaki tüm `ciadpi` süreçlerini öldürür.** Çalışan bir oturum varken genel `pkill ciadpi` çalıştırma; testler `isolate(t)` ile `HOME`'u geçici dizine alır, uygulama `SingleInstanceLock` kullanır.
- `ciadpi` varsayılan olarak `0.0.0.0` dinler (`-i 127.0.0.1` zorunlu) ve SIGTERM'i yok sayar (`Kill` kullanılır).
- Sistem DNS'i engelli; proxy tarafında da DNS çözümü çalışmaz. Önyüz Discord alan adlarını `1.1.1.1` üzerinden DoH ile çözer.
- `networksetup -setautoproxyurl` boş URL kabul etmez (sistem proxy yaklaşımı bu yüzden ve gereksiz olduğu için kaldırıldı; geri getirme).
- `gosec`, Go 1.27.2 altında tip yükleme hatası verir (G104 taranamaz); yok sayılan hatalar elle `_ =` ile işaretlenir, yanlış alarmlar gerekçeli `#nosec` taşır.
- `ciadpi.exe` her derlemede farklı hash üretir (PE zaman damgası); macOS ikilisi tekrarlanabilirdir.
- Dağıtım: imza ad-hoc, Gatekeeper `rejected` der. Uyarısız dağıtım için Developer ID + notarization gerekir. `LSMinimumSystemVersion` 13.0.0 (Go ikilisi macOS 13 nesneleriyle bağlanır). `hdiutil` macOS 27'de "deprecated" uyarısı verir ama çalışır.
