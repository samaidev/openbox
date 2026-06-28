#!/bin/bash
# OpenBox — macOS .pkg build script
# ----------------------------------------------------------------------------
# Produces dist/OpenBox-<version>-macos.pkg from a clean checkout.
#
# Pipeline:
#   1. go build the darwin binary (current arch; universal if both are avail)
#   2. assemble OpenBox.app bundle with Info.plist + icon.icns + LICENSE
#   3. ad-hoc codesign (required for Apple Silicon to run at all)
#   4. pkgbuild the .app into a component package
#   5. productbuild wraps it with welcome/readme/license/conclusion UI
#   6. optionally productsign with APPLE_DEVELOPER_ID_INSTALLER if set
#
# Usage:
#   bash build/macos/build.sh                 # uses VERSION from env or 0.1.0
#   VERSION=0.2.0 bash build/macos/build.sh
#
# Prereqs (only on macOS):
#   • Go 1.23+
#   • Xcode Command Line Tools (for iconutil, sips, pkgbuild, productbuild)
#
# Optional env vars:
#   • VERSION                       — version string (default: 0.1.0)
#   • APPLE_DEVELOPER_ID_INSTALLER  — Developer ID Installer cert name; if
#                                     set, the .pkg is signed with productsign
#   • NOTARY_PROFILE                — notarytool profile name; if set, the
#                                     .pkg is submitted for notarization
# ----------------------------------------------------------------------------

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-0.1.0}"
APP_NAME="OpenBox"
APP_ID="io.github.samaidev.openbox"
APP_DIR="dist/${APP_NAME}.app"
PKGROOT="dist/pkgroot"
MACOS_BUILD="build/macos"

echo "==> Building ${APP_NAME} ${VERSION} for darwin/$(uname -m)"

# ---------------------------------------------------------------------------
# 1. Compile the Go binary
# ---------------------------------------------------------------------------
echo "==> [1/6] go build"
mkdir -p "${APP_DIR}/Contents/MacOS" "${APP_DIR}/Contents/Resources"

CGO_ENABLED=1 go build \
    -trimpath \
    -ldflags "-X main.Version=${VERSION}" \
    -o "${APP_DIR}/Contents/MacOS/openbox" \
    ./cmd/openbox

# ---------------------------------------------------------------------------
# 2. Assemble the .app bundle
# ---------------------------------------------------------------------------
echo "==> [2/6] assembling .app bundle"
cp "${MACOS_BUILD}/Info.plist" "${APP_DIR}/Contents/Info.plist"
cp LICENSE README.md "${APP_DIR}/Contents/Resources/"

# Generate .icns from the master 1024x1024 PNG using sips + iconutil.
ICONSET="${APP_DIR}/Contents/Resources/icon.iconset"
mkdir -p "$ICONSET"
MASTER_PNG="internal/assets/icon.png"
for size in 16 32 64 128 256 512; do
    double=$((size * 2))
    sips -z "$size" "$size" "$MASTER_PNG" --out "${ICONSET}/icon_${size}x${size}.png" >/dev/null
    if [ "$double" -le 1024 ]; then
        sips -z "$double" "$double" "$MASTER_PNG" --out "${ICONSET}/icon_${size}x${size}@2x.png" >/dev/null
    fi
done
# 1024x1024 (no @2x needed; master is already 1024)
cp "$MASTER_PNG" "${ICONSET}/icon_512x512@2x.png"
iconutil -c icns "$ICONSET" -o "${APP_DIR}/Contents/Resources/icon.icns"
rm -rf "$ICONSET"

# ---------------------------------------------------------------------------
# 3. Ad-hoc codesign (mandatory on Apple Silicon; harmless on Intel)
# ---------------------------------------------------------------------------
echo "==> [3/6] ad-hoc codesign"
codesign --force --deep --sign - "${APP_DIR}" 2>&1 | sed 's/^/    /' || true

# ---------------------------------------------------------------------------
# 4. pkgbuild component package
# ---------------------------------------------------------------------------
echo "==> [4/6] pkgbuild"
rm -rf "$PKGROOT"
mkdir -p "${PKGROOT}/Applications"
cp -R "${APP_DIR}" "${PKGROOT}/Applications/"

COMPONENT_PKG="dist/OpenBox-${VERSION}-component.pkg"
pkgbuild \
    --root "$PKGROOT" \
    --identifier "$APP_ID" \
    --version "$VERSION" \
    --install-location "/" \
    --scripts "${MACOS_BUILD}/scripts" \
    "$COMPONENT_PKG" 2>&1 | sed 's/^/    /'

# ---------------------------------------------------------------------------
# 5. productbuild — wrap with welcome/readme/license/conclusion
# ---------------------------------------------------------------------------
echo "==> [5/6] productbuild"
FINAL_PKG="dist/OpenBox-${VERSION}-macos.pkg"
productbuild \
    --distribution "${MACOS_BUILD}/Distribution.xml" \
    --resources "${MACOS_BUILD}/resources" \
    --package-path "dist" \
    "$FINAL_PKG" 2>&1 | sed 's/^/    /'

# Remove the intermediate component package; only the final .pkg ships.
rm -f "$COMPONENT_PKG"

# ---------------------------------------------------------------------------
# 6. Optional: sign with Developer ID Installer + notarize
# ---------------------------------------------------------------------------
if [ -n "${APPLE_DEVELOPER_ID_INSTALLER:-}" ]; then
    echo "==> [6/6] productsign with Developer ID"
    SIGNED="dist/OpenBox-${VERSION}-macos-signed.pkg"
    productsign --sign "$APPLE_DEVELOPER_ID_INSTALLER" "$FINAL_PKG" "$SIGNED"
    mv "$SIGNED" "$FINAL_PKG"

    if [ -n "${NOTARY_PROFILE:-}" ]; then
        echo "==> notarizing with notarytool (profile: ${NOTARY_PROFILE})"
        xcrun notarytool submit "$FINAL_PKG" \
            --keychain-profile "$NOTARY_PROFILE" \
            --wait
        echo "==> stapling ticket"
        xcrun stapler staple "$FINAL_PKG" || true
    fi
else
    echo "==> [6/6] skipping Developer ID sign (set APPLE_DEVELOPER_ID_INSTALLER to enable)"
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
SIZE=$(du -h "$FINAL_PKG" | cut -f1)
echo ""
echo "==> Built: ${FINAL_PKG}  (${SIZE})"
echo ""
echo "To install: double-click the .pkg, or run:"
echo "    sudo installer -pkg ${FINAL_PKG} -target /"
