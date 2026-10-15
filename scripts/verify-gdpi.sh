#!/bin/sh
# Gomulu GoodbyeDPI-Turkey ikililerini resmi surum paketiyle karsilastirir (THIRD_PARTY.md).
set -eu
URL="https://github.com/cagritaskn/GoodbyeDPI-Turkey/releases/download/release-0.2.3rc3-turkey/goodbyedpi-0.2.3rc3-turkey.zip"
ZIP_SHA="b1f93b2e9434d93c5321275c4a3d0a87f3b822c552eceabdbeb1610c879e1863"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

curl -fsSL -o "$WORK/rel.zip" "$URL"
GOT="$(shasum -a 256 "$WORK/rel.zip" | cut -d' ' -f1)"
[ "$GOT" = "$ZIP_SHA" ] || { echo "HATA: zip ozeti uyusmuyor ($GOT)" >&2; exit 1; }

for f in goodbyedpi.exe WinDivert.dll WinDivert64.sys; do
  A="$(unzip -p "$WORK/rel.zip" "x86_64/$f" | shasum -a 256 | cut -d' ' -f1)"
  B="$(shasum -a 256 "$ROOT/engine/gdpi_windows/$f" | cut -d' ' -f1)"
  [ "$A" = "$B" ] || { echo "HATA: $f farkli (paket $A, depo $B)" >&2; exit 1; }
  echo "OK $f $A"
done
