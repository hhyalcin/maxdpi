# Üçüncü taraf bileşenler

MaxDPI aşağıdaki açık kaynak bileşenleri içerir veya derleme sırasında kullanır. Lisans metinleri
`licenses/third_party/` altındadır. MaxDPI'ın kendi lisansı MIT'dir (`LICENSE`).

| Bileşen | Sürüm | Lisans | Kaynak | Nerede |
|---|---|---|---|---|
| ByeDPI (`ciadpi`) | v0.17.3 (`7efde1b1296eaaa187b70e951894dde17527489c`) | MIT | https://github.com/hufrea/byedpi | macOS motoru; `scripts/build-engines.sh` kaynaktan derler, depoda yok |
| GoodbyeDPI-Turkey | release-0.2.3rc3-turkey (GoodbyeDPI 0.2.3rc3 çatalı) | Apache-2.0 | https://github.com/cagritaskn/GoodbyeDPI-Turkey (üst depo: https://github.com/ValdikSS/GoodbyeDPI) | Windows motoru; `engine/gdpi_windows/goodbyedpi.exe` |
| WinDivert | 2.2 | LGPL-3.0 | https://github.com/basil00/WinDivert | Windows; `WinDivert.dll`, `WinDivert64.sys` |
| getline, uthash | GoodbyeDPI içinden | BSD | GoodbyeDPI kaynağı | `licenses/third_party/` |

## Gömülü Windows ikilileri

`engine/gdpi_windows/` altındaki üç dosya, aşağıdaki sürüm paketindeki `x86_64/` dizininden **değiştirilmeden**
alınmıştır:

https://github.com/cagritaskn/GoodbyeDPI-Turkey/releases/download/release-0.2.3rc3-turkey/goodbyedpi-0.2.3rc3-turkey.zip
(zip SHA-256: `b1f93b2e9434d93c5321275c4a3d0a87f3b822c552eceabdbeb1610c879e1863`, 2026-10-16'da indirildi)

| Dosya | SHA-256 |
|---|---|
| `goodbyedpi.exe` | `8d412b094bb9c137ff25ba9a794d1122ecc84bb776debff6c249723a13cc31cd` |
| `WinDivert.dll` | `6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e` |
| `WinDivert64.sys` | `e69b5ba3f0cd6cfb2983e442636e7f0b342b61b15264b0328317d4559c82cf50` |

Doğrulama: `scripts/verify-gdpi.sh` sürüm paketini indirir ve özetleri karşılaştırır; `go test ./engine`
gömülü dosyaların özetlerini kontrol eder.

Sınırlar: bu ikililer bu depoda **kaynaktan derlenmez**; güven, çatalın sürüm paketine dayanır (paket, üst
depodaki GoodbyeDPI ve WinDivert'in resmi ikilileriyle karşılaştırılmadı; yalnızca `WinDivert64.sys`
aynı paketin `x86/` dizinindeki kopyasıyla eşleşiyor). Tam güven için ikilileri kaynaktan kendiniz derleyip
`engine/gdpi_windows/` altına koyabilirsiniz.

## LGPL-3.0 (WinDivert)

WinDivert dinamik kütüphane (`WinDivert.dll`) ve sürücü olarak, ayrı dosyalar halinde ve değiştirilmeden
dağıtılır. Kaynak kodu: https://github.com/basil00/WinDivert. Dosyaları kendi derlemenizle değiştirmek için
`engine/gdpi_windows/` altındaki dosyaları değiştirip uygulamayı yeniden derleyebilirsiniz (hash testini de
güncelleyin).
