#!/bin/sh
# DPI motorlarini sabitlenmis commit'ten kaynaktan derler (hazir binary indirilmez).
# Kullanim: scripts/build-engines.sh   (macOS: ciadpi, Windows: ciadpi.exe -> engine/bin_*)
set -eu

BYEDPI_REPO="https://github.com/hufrea/byedpi.git"
BYEDPI_TAG="v0.17.3"
BYEDPI_COMMIT="7efde1b1296eaaa187b70e951894dde17527489c"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

git clone -q --depth 1 --branch "$BYEDPI_TAG" "$BYEDPI_REPO" "$WORK/byedpi"
GOT="$(git -C "$WORK/byedpi" rev-parse HEAD)"
if [ "$GOT" != "$BYEDPI_COMMIT" ]; then
  echo "HATA: byedpi commit uyusmuyor (beklenen $BYEDPI_COMMIT, gelen $GOT)" >&2
  exit 1
fi

# macOS: yerel derleme
if [ "$(uname -s)" = "Darwin" ]; then
  make -C "$WORK/byedpi" >/dev/null
  mkdir -p "$ROOT/engine/bin_darwin"
  cp "$WORK/byedpi/ciadpi" "$ROOT/engine/bin_darwin/ciadpi"
  chmod 0755 "$ROOT/engine/bin_darwin/ciadpi"
  shasum -a 256 "$ROOT/engine/bin_darwin/ciadpi"
fi

# Windows: mingw-w64 ile capraz derleme (statik; yalnizca sistem DLL'lerine baglanir)
if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
  make -C "$WORK/byedpi" clean >/dev/null
  make -C "$WORK/byedpi" windows CC=x86_64-w64-mingw32-gcc \
    WIN_LDFLAGS="-lws2_32 -lmswsock -static" >/dev/null
  mkdir -p "$ROOT/engine/bin_windows"
  cp "$WORK/byedpi/ciadpi.exe" "$ROOT/engine/bin_windows/ciadpi.exe"
  shasum -a 256 "$ROOT/engine/bin_windows/ciadpi.exe"
else
  echo "UYARI: mingw-w64 yok, Windows motoru derlenmedi" >&2
fi

echo "byedpi $BYEDPI_TAG $BYEDPI_COMMIT"
