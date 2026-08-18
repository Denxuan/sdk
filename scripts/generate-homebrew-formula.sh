#!/usr/bin/env bash

# Generate a Homebrew Formula that installs SDK's prebuilt GitHub Release assets.
# Usage: ./scripts/generate-homebrew-formula.sh v0.0.2 [output-path]

set -euo pipefail

TAG="${1:-}"
if [[ -z "$TAG" ]]; then
  echo "Usage: $0 <version> [output-path], for example: $0 v0.0.2" >&2
  exit 1
fi

RELEASE_TAG="$TAG"
VERSION="${RELEASE_TAG#v}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist/v$VERSION"
CHECKSUMS_FILE="$DIST_DIR/checksums.txt"
OUTPUT_FILE="${2:-$DIST_DIR/sdk.rb}"
REPOSITORY="${GITHUB_REPOSITORY:-Denxuan/sdk}"
RELEASE_URL="https://github.com/$REPOSITORY/releases/download/$RELEASE_TAG"

if [[ ! -f "$CHECKSUMS_FILE" ]]; then
  echo "Checksums file not found: $CHECKSUMS_FILE" >&2
  echo "Run ./scripts/package-release.sh $RELEASE_TAG first." >&2
  exit 1
fi

checksum_for() {
  local file_name="$1"
  local checksum
  checksum="$(awk -v name="$file_name" '$2 == name { print $1 }' "$CHECKSUMS_FILE")"
  if [[ -z "$checksum" ]]; then
    echo "Checksum not found for: $file_name" >&2
    exit 1
  fi
  printf '%s' "$checksum"
}

darwin_amd64="$(checksum_for "sdk_${VERSION}_darwin_amd64.tar.gz")"
darwin_arm64="$(checksum_for "sdk_${VERSION}_darwin_arm64.tar.gz")"
linux_amd64="$(checksum_for "sdk_${VERSION}_linux_amd64.tar.gz")"
linux_arm64="$(checksum_for "sdk_${VERSION}_linux_arm64.tar.gz")"

mkdir -p "$(dirname "$OUTPUT_FILE")"
cat > "$OUTPUT_FILE" <<EOF
class Sdk < Formula
  desc "Development tool version manager for Java, Maven, Maven mvnd, Gradle, Go, and Node.js"
  homepage "https://github.com/$REPOSITORY"
  version "$VERSION"

  on_macos do
    on_arm do
      url "$RELEASE_URL/sdk_#{version}_darwin_arm64.tar.gz"
      sha256 "$darwin_arm64"
    end

    on_intel do
      url "$RELEASE_URL/sdk_#{version}_darwin_amd64.tar.gz"
      sha256 "$darwin_amd64"
    end
  end

  on_linux do
    on_arm do
      url "$RELEASE_URL/sdk_#{version}_linux_arm64.tar.gz"
      sha256 "$linux_arm64"
    end

    on_intel do
      url "$RELEASE_URL/sdk_#{version}_linux_amd64.tar.gz"
      sha256 "$linux_amd64"
    end
  end

  def install
    bin.install "sdk"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/sdk version")
  end
end
EOF

echo "Homebrew Formula created: $OUTPUT_FILE"
