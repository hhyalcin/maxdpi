#!/bin/sh
# build/bin/maxdpi.app'den surukle-birak kurulumlu .dmg uretir.
# Once: scripts/build-engines.sh && wails build -platform darwin/universal
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/build/bin/maxdpi.app"
[ -d "$APP" ] || { echo "HATA: $APP yok (once wails build)" >&2; exit 1; }

VER="$(plutil -extract CFBundleShortVersionString raw "$APP/Contents/Info.plist")"
OUT="$ROOT/build/bin/MaxDPI-$VER-macOS.dmg"

# Imza bozuksa paketleme (kopyalama sonrasi da dogrulanir)
codesign --verify --deep --strict "$APP"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
ditto "$APP" "$STAGE/maxdpi.app"          # ditto: imza ve genisletilmis ozellikleri korur
ln -s /Applications "$STAGE/Applications"  # surukle-birak hedefi
codesign --verify --deep --strict "$STAGE/maxdpi.app"

rm -f "$OUT"
hdiutil create -volname "MaxDPI" -srcfolder "$STAGE" -fs HFS+ -format UDZO -ov "$OUT" >/dev/null
echo "$OUT"
shasum -a 256 "$OUT"
