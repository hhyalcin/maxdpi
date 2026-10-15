# Güvenlik politikası

MaxDPI, Windows'ta yönetici izniyle bir çekirdek sürücüsü (WinDivert) yükler; bu yüzden güvenlik raporlarını ciddiye alırız.

## Açık bildirme

Lütfen güvenlik açıklarını **herkese açık issue olarak açmayın**. GitHub'daki
**Security > Report a vulnerability** (özel güvenlik danışmanlığı) bölümünü kullanın.

Rapora şunları ekleyin: etkilenen sürüm, işletim sistemi, yeniden üretme adımları ve etkisi.

Beklenti: raporu en geç 7 gün içinde yanıtlamayı hedefleriz; bu bir SLA değildir, tek kişilik bir projedir.

## Kapsam

- Yetki yükseltme (özellikle Windows motor dosyaları, çıkarma dizini, Discord başlatma)
- Yerel önyüzün (macOS, `127.0.0.1`) kötüye kullanımı
- Gömülü üçüncü taraf ikililerin bütünlüğü (bkz. THIRD_PARTY.md)

Kapsam dışı: GoodbyeDPI, WinDivert, ByeDPI veya Discord'un kendi açıkları (ilgili projelere bildirin);
uygulamanın imzasız olması; antivirüs yanlış alarmları.
