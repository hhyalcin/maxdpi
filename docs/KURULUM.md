# MaxDPI Kurulum ve İzin Rehberi

MaxDPI iki platformda farklı çalışır:
- **macOS:** yönetici yetkisi istemez, sistem proxy'sine dokunmaz. Yerel önyüz yalnızca `127.0.0.1` dinler.
- **Windows:** gömülü GoodbyeDPI + WinDivert sürücüsünü kullanır (paket seviyesinde, sistem geneli). **Yönetici izni ister**, ayrıca bir şey kurman gerekmez.

Uygulama imzasızdır; aşağıdaki adımlar işletim sistemi ve antivirüs uyarılarını aşmak içindir.

Kurallar (her iki sistemde):
- macOS'ta uygulamayı `sudo` ile çalıştırma. Gerekmez.
- Güvenlik özelliğini tümden kapatma (`spctl --master-disable`, Defender'ı kapatmak vb.). İstisnayı yalnızca MaxDPI'a ver.
- Dosyayı yalnızca resmi kaynaktan indir ve yayımlanan SHA-256 özetiyle karşılaştır.
- Discord'u başlatmadan önce tamamen kapat (Windows'ta sistem tepsisi dahil).

---

## A. Kullanıcı rehberi

### macOS (13 Ventura ve üstü)

| # | Adım | Ne görürsün | Ne yaparsın |
|---|------|-------------|-------------|
| 1 | İndir | `MaxDPI-x.y.z-macOS.dmg` | Özeti doğrula: `shasum -a 256 ~/Downloads/MaxDPI-*.dmg` |
| 2 | Kur | DMG açılır | `maxdpi.app` dosyasını `Applications` klasörüne sürükle |
| 3 | İlk açılış | "Apple, kötü amaçlı yazılım içermediğini doğrulayamadı" veya "hasarlı" | Uygulama henüz imzalı/notarize değil. Adım 4'e geç |
| 4 | İzin ver | | **Sistem Ayarları > Gizlilik ve Güvenlik** altında aşağı in, MaxDPI için **Yine de Aç** düğmesine bas, parolayı/Touch ID'yi onayla |
| 4b | Alternatif | Düğme çıkmazsa | Terminal: `xattr -dr com.apple.quarantine /Applications/maxdpi.app` (yalnızca bu uygulama için karantina bayrağını kaldırır) |
| 5 | Güvenlik duvarı | "Gelen bağlantılara izin ver?" çıkarsa | Önyüz yalnız `127.0.0.1` dinler, dışarıya açık değildir. Çıkarsa **İzin Ver** de |
| 6 | Kullan | | Discord'u kapat, MaxDPI'da **Bağlan**, ardından Discord'u MaxDPI'dan aç |

macOS 15 (Sequoia) ve sonrasında sağ tık > Aç yöntemi artık işe yaramaz; yalnız adım 4/4b kullanılır.

İstenmeyecek izinler (istenirse sorun var demektir): Erişilebilirlik, Tam Disk Erişimi, Kamera/Mikrofon, Yönetici parolası (adım 4 hariç).

### Windows (10 / 11)

| # | Adım | Ne görürsün | Ne yaparsın |
|---|------|-------------|-------------|
| 1 | İndir | `MaxDPI-x.y.z-windows-amd64.zip` | Özeti doğrula: PowerShell `Get-FileHash .\MaxDPI-*-windows-amd64.zip -Algorithm SHA256` |
| 2 | Aç | Zip içinde `maxdpi.exe` | Zip'i bir klasöre çıkar (ör. `Belgeler\MaxDPI`). Zip'in içinden doğrudan çalıştırma |
| 3 | SmartScreen | "Windows bilgisayarınızı korudu" | **Ek bilgi > Yine de çalıştır** (uygulama imzasız) |
| 3b | Alternatif | Mavi ekran hiç çıkmadan engellenirse | Dosya > Sağ tık > Özellikler > en altta **Engellemeyi kaldır** > Uygula. Ya da: `Unblock-File .\maxdpi.exe` |
| 4 | Yönetici izni | UAC: "Bu uygulamanın cihazınızda değişiklik yapmasına izin vermek istiyor musunuz?" (yayıncı: Bilinmeyen) | **Evet** de. Paket filtresi (WinDivert) sürücüsü için zorunludur, izin vermezsen Bağlan çalışmaz |
| 5 | Antivirüs | Defender "Tehdit bulundu"; `WinDivert64.sys` veya `goodbyedpi.exe` karantinaya alındı | Paket yakalayan sürücüler sık yanlış pozitif verir. Dosyayı **özetle doğruladıysan** karantinadan **geri yükle** ve yalnız şu klasörü dışlamaya ekle: `%LOCALAPPDATA%\maxdpi` (**Windows Güvenliği > Virüs ve tehdit koruması > Ayarları yönet > Dışlamalar**). Tüm sürücüyü dışlama. Kaspersky ile çalışmaz; farklı antivirüs kullan veya MaxDPI'ı dışla |
| 6 | WebView2 | "WebView2 Runtime gerekli" | Win 11'de hazırdır. Win 10'da Microsoft'un "Evergreen Bootstrapper" kurulumunu çalıştır |
| 7 | Kullan | | Discord'u tamamen kapat (tepsideki simge dahil), MaxDPI'da **Bağlan**, ardından Discord'u MaxDPI'dan aç |

Notlar:
- Başka bir GoodbyeDPI / WinDivert / zapret çalışıyorsa çakışır. Önce onları kapat. MaxDPI kendi `goodbyedpi.exe` süreçlerini temizler.
- **Bağlan** birkaç strateji dener, 1 dakikayı bulabilir. Hata verirse mesajdaki "son hata" bilgisini ve günlükleri paylaş: `MAXDPI_DEBUG=1` ile çalıştırınca `%LOCALAPPDATA%\maxdpi\debug.log`; pencere hiç açılmazsa aynı klasördeki `startup.log`.
- Windows'ta Discord, MaxDPI bağlıyken proxy'siz açılır; sistem proxy'si değiştirilmez.
- Kurumsal/okul bilgisayarı: AppLocker veya Grup İlkesi `%LOCALAPPDATA%` altındaki exe/sürücüleri engelleyebilir. Bu durumda BT yöneticisiyle görüş; kendin atlatmaya çalışma.

---

## B. Geliştirici yol haritası (kurulumu kolaylaştırmak için)

Sıra önemli.

### Faz 0 — Windows'ta gerçek doğrulama
- [x] Gerçek hatta Bağlan, Discord güncelleyicisi, giriş ve ses doğrulandı (2026-10-13, GoodbyeDPI motoru).
- [ ] Windows 10 ve 11'de ayrı ayrı dene; Defender/SmartScreen/UAC ekranlarını kaydet, yukarıdaki tablo gerçek ekranlarla düzeltilsin.
- [ ] Farklı ISS'lerde `DefaultProfiles` sıralamasını doğrula.

### Faz 1 — Uygulama içi "Kurulum Sağlığı" ekranı (kullanıcıya en büyük kazanç)
İlk açılışta ve Bağlan hatasında kontrol et, her başarısızlık için tek satır açıklama + "Nasıl düzeltilir" bağlantısı göster:
- [ ] Discord kurulu mu? (`updateExe()` / macOS'ta `/Applications/Discord.app`)
- [ ] Discord açık mı? (`DiscordRunning()`) — "Kapat ve tekrar dene" mesajı.
- [ ] Motor çıkarıldı ve başlatılabildi mi? Windows'ta `goodbyedpi.exe`/`WinDivert64.sys` yazılamıyor/başlamıyorsa "antivirüs engellemiş olabilir" + dışlama adımı (A.4) ve dosya yolu.
- [ ] DoH ön kontrolü (`App.Connect` zaten yapıyor): başarısızsa "İnternet/1.1.1.1 erişimi yok" ayrımı.
- [ ] Hata metinlerini ASCII'den çıkarıp Türkçe karakterli göster (`"Discord acik"` vb. arayüzde).
- [ ] "Tanı raporunu kopyala" düğmesi: işletim sistemi sürümü, hangi kontrol başarısız, `debug.log` son satırları (kişisel veri/IP içermeyen).

### Faz 2 — Dağıtım güveni (uyarıların asıl çözümü)
- [ ] macOS: Apple Developer ID imzası + notarization + staple. Sonuç: Gatekeeper uyarısı kalkar, A.4 adımı gerekmez.
- [ ] Windows: kod imzalama. Şimdilik **imzasız** dağıtılıyor (karar: 2026-10-15); ücretsiz yol yalnız açık kaynak için SignPath Foundation, ücretli yol Azure Trusted Signing / OV-EV sertifika.
- [ ] Windows kurulumu: şu an yalnız zip içinde `maxdpi.exe`. Wails NSIS installer (`build/windows/installer`) üretilirse manifest yönetici izni istediği için kurulum da yönetici gerektirir; WebView2 bootstrapper dahil.
- [ ] Her sürümde `SHA256SUMS` yayımla; indirme sayfasında doğrulama komutları (A tablolarındaki 1. adım).

### Faz 3 — Antivirüs itibarı
- [ ] Microsoft Defender için yanlış pozitif bildirimi (Microsoft Security Intelligence "Submit a file"); `WinDivert64.sys` ve `goodbyedpi.exe` için ayrı gönder.
- [ ] README ve indirme sayfasında "neden uyarı çıkıyor" kısa açıklaması: GoodbyeDPI ve WinDivert açık kaynak (lisanslar `licenses/third_party/`); macOS'ta ByeDPI, derleme betiği `scripts/build-engines.sh`.

### Faz 4 — Yardım içeriği
- [ ] Bu rehberin ekran görüntülü kısa sürümü uygulama içinde "Yardım" sekmesinde (işletim sistemine göre yalnız ilgili bölüm).
- [ ] README'ye bu dosyaya bağlantı.

### Güvenlik notları (tasarım kısıtı)
- macOS uygulaması yönetici yetkisi istememeli. Windows'ta yönetici izni WinDivert sürücüsü için zorunlu (manifest `requireAdministrator`); bunun dışında hiçbir özellik ek yetki istememeli.
- Dışlama/izin adımlarını uygulama kendisi yapmamalı (Defender dışlamasını otomatik eklemek yetki yükseltme ve kötüye kullanım deseni). Yalnızca kullanıcıya göster.
- Yardım bağlantıları `App.OpenLink(id)` sabit tablosuna eklenmeli; arayüz keyfi URL açmamalı.
