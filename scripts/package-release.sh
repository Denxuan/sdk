#!/usr/bin/env bash

# Build release archives for every supported platform.
# Usage: ./scripts/package-release.sh v0.0.2

set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>, for example: $0 v0.0.2" >&2
  exit 1
fi

VERSION="${VERSION#v}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist/v$VERSION"
MODULE_PATH="$(cd "$ROOT_DIR" && go list -m -f '{{.Path}}')"
LDFLAGS="-s -w -X ${MODULE_PATH}/internal/buildinfo.Version=${VERSION}"

if [[ -e "$DIST_DIR" ]]; then
  echo "Release directory already exists: $DIST_DIR" >&2
  echo "Choose a new version or remove the existing directory first." >&2
  exit 1
fi

mkdir -p "$DIST_DIR"

build_target() {
  local operating_system="$1"
  local architecture="$2"
  local binary_name="sdk"
  local archive_name="sdk_${VERSION}_${operating_system}_${architecture}"
  local stage_dir="$DIST_DIR/$archive_name"

  if [[ "$operating_system" == "windows" ]]; then
    binary_name="sdk.exe"
  fi

  mkdir -p "$stage_dir"
  echo "Building $operating_system/$architecture..."
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$operating_system" GOARCH="$architecture" \
      go build -trimpath -ldflags "$LDFLAGS" -o "$stage_dir/$binary_name" .
  )

  if [[ "$operating_system" == "windows" ]]; then
    (
      cd "$DIST_DIR"
      zip -qr "${archive_name}.zip" "$archive_name"
    )
  else
    tar -C "$DIST_DIR" -czf "$DIST_DIR/${archive_name}.tar.gz" "$archive_name"
  fi

  rm -rf "$stage_dir"
}

build_target darwin amd64
build_target darwin arm64
build_target linux amd64
build_target linux arm64
build_target windows amd64

(
  cd "$DIST_DIR"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 sdk_* > checksums.txt
  else
    sha256sum sdk_* > checksums.txt
  fi
)

echo
echo "Release archives created in: $DIST_DIR"
echo "Upload every archive and checksums.txt to GitHub Release v$VERSION."
