# MaxDPI Kurulum ve İzin Rehberi

MaxDPI yönetici (admin/root) yetkisi istemez, sistem proxy'sine ve güvenlik ayarlarına dokunmaz.
Yerel önyüz yalnızca `127.0.0.1` dinler. Aşağıdaki adımlar, uygulamanın imzasız olmasından
(işletim sistemi uyarıları) ve antivirüs yazılımlarından kaynaklanan engelleri aşmak içindir.

Kurallar (her iki sistemde):
- Uygulamayı "yönetici olarak çalıştır" / `sudo` ile çalıştırma. Gerekmez.
- Güvenlik özelliğini tümden kapatma (`spctl --master-disable`, Defender'ı kapatmak vb.). İstisnayı yalnızca MaxDPI'a ver.
- Dosyayı yalnızca resmi kaynaktan indir ve yayımlanan SHA-256 özetiyle karşılaştır.
- Discord'u başlatmadan önce tamamen kapat (Windows'ta sistem tepsisi dahil). Açık Discord'a proxy uygulanamaz.

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
| 1 | İndir | `MaxDPI-Setup-x.y.z.exe` | Özeti doğrula: PowerShell `Get-FileHash .\MaxDPI-Setup-*.exe -Algorithm SHA256` |
| 2 | SmartScreen | "Windows bilgisayarınızı korudu" | **Ek bilgi > Yine de çalıştır** (imzalı sürümde bu ekran çıkmaz) |
| 2b | Alternatif | Mavi ekran hiç çıkmadan engellenirse | Dosya > Sağ tık > Özellikler > en altta **Engellemeyi kaldır** > Uygula. Ya da: `Unblock-File .\MaxDPI-Setup-*.exe` |
| 3 | Kur | Kurulum sihirbazı | Yönetici istemez; yalnız kendi kullanıcın için kurulur |
| 4 | Antivirüs | Defender "Tehdit bulundu" / `ciadpi.exe` karantinaya alındı | DPI atlatma araçları sık yanlış pozitif verir. Dosyayı **özetle doğruladıysan** karantinadan **geri yükle** ve yalnız şu klasörü dışlamaya ekle: `%LOCALAPPDATA%\maxdpi` (**Windows Güvenliği > Virüs ve tehdit koruması > Ayarları yönet > Dışlamalar**). Tüm sürücüyü dışlama |
| 5 | Güvenlik duvarı | "Windows Defender Güvenlik Duvarı bazı özellikleri engelledi" | **Yalnızca Özel ağlar**'ı işaretle, **Genel ağlar**'ı işaretleme. Önyüz zaten yalnız `127.0.0.1` dinler |
| 6 | WebView2 | "WebView2 Runtime gerekli" | Win 11'de hazırdır. Win 10'da kurulum bunu indirir; olmazsa Microsoft'un "Evergreen Bootstrapper" kurulumunu çalıştır |
| 7 | Kullan | | Discord'u tamamen kapat (tepsideki simge dahil), MaxDPI'da **Bağlan**, ardından Discord'u MaxDPI'dan aç |

Kurumsal/okul bilgisayarı: AppLocker veya Grup İlkesi `%LOCALAPPDATA%` altındaki exe'leri engelleyebilir. Bu durumda BT yöneticisiyle görüş; kendin atlatmaya çalışma.

---

## B. Geliştirici yol haritası (kurulumu kolaylaştırmak için)

Sıra önemli: 0 tamamlanmadan Windows için vaat verme.

### Faz 0 — Windows'ta gerçek doğrulama (engelleyici)
- [ ] Gerçek bir Windows 10 ve 11 makinesinde `maxdpi.exe`: Bağlan, Discord girişi, mesaj, sesli kanal.
- [ ] `ciadpi.exe` ilk çalıştırmada Defender/SmartScreen/duvar davranışını kaydet; yukarıdaki tablo gerçek ekranlarla düzeltilsin.
- [ ] `DefaultProfiles` sıralamasını gerçek hatta doğrula (`byedpi_windows.go` yorumundaki "gerçek test gerekir").

### Faz 1 — Uygulama içi "Kurulum Sağlığı" ekranı (kullanıcıya en büyük kazanç)
İlk açılışta ve Bağlan hatasında kontrol et, her başarısızlık için tek satır açıklama + "Nasıl düzeltilir" bağlantısı göster:
- [ ] Discord kurulu mu? (`updateExe()` / macOS'ta `/Applications/Discord.app`)
- [ ] Discord açık mı? (`DiscordRunning()`) — "Kapat ve tekrar dene" mesajı.
- [ ] Motor çıkarıldı ve başlatılabildi mi? Windows'ta `ciadpi.exe` yazılamıyor/başlamıyorsa "antivirüs engellemiş olabilir" + dışlama adımı (A.4) ve dosya yolu.
- [ ] DoH ön kontrolü (`App.Connect` zaten yapıyor): başarısızsa "İnternet/1.1.1.1 erişimi yok" ayrımı.
- [ ] Hata metinlerini ASCII'den çıkarıp Türkçe karakterli göster (`"Discord acik"` vb. arayüzde).
- [ ] "Tanı raporunu kopyala" düğmesi: işletim sistemi sürümü, hangi kontrol başarısız, `debug.log` son satırları (kişisel veri/IP içermeyen).

### Faz 2 — Dağıtım güveni (uyarıların asıl çözümü)
- [ ] macOS: Apple Developer ID imzası + notarization + staple. Sonuç: Gatekeeper uyarısı kalkar, A.4 adımı gerekmez.
- [ ] Windows: kod imzalama (EV sertifika ya da Azure Trusted Signing). Sonuç: SmartScreen itibarı oluşur.
- [ ] `ciadpi.exe` her derlemede farklı hash üretiyor (PE zaman damgası). Antivirüs itibarı ve izin listesi için tekrarlanabilir derleme yap (mingw bağlayıcıya `-Wl,--no-insert-timestamp`).
- [ ] Windows kurulumu: Wails NSIS (`build/windows/installer`) kullanıcı bazlı kurulum (`RequestExecutionLevel user`), WebView2 bootstrapper dahil.
- [ ] Her sürümde `SHA256SUMS` yayımla; indirme sayfasında doğrulama komutları (A tablolarındaki 1. adım).

### Faz 3 — Antivirüs itibarı
- [ ] Microsoft Defender için yanlış pozitif bildirimi (Microsoft Security Intelligence "Submit a file"); imzalı + tekrarlanabilir sürümle gönder.
- [ ] README ve indirme sayfasında "neden uyarı çıkıyor" kısa açıklaması: ByeDPI kaynak kodu açık, MIT; derleme betiği `scripts/build-engines.sh`.

### Faz 4 — Yardım içeriği
- [ ] Bu rehberin ekran görüntülü kısa sürümü uygulama içinde "Yardım" sekmesinde (işletim sistemine göre yalnız ilgili bölüm).
- [ ] README'ye bu dosyaya bağlantı.

### Güvenlik notları (tasarım kısıtı)
- Uygulama hiçbir aşamada yönetici yetkisi istememeli. Windows manifestinde `requestedExecutionLevel` ayarsız (asInvoker) kalmalı; bir öneri yönetici gerektiriyorsa reddet.
- Dışlama/izin adımlarını uygulama kendisi yapmamalı (Defender dışlamasını otomatik eklemek yetki yükseltme ve kötüye kullanım deseni). Yalnızca kullanıcıya göster.
- Yardım bağlantıları `App.OpenLink(id)` sabit tablosuna eklenmeli; arayüz keyfi URL açmamalı.
