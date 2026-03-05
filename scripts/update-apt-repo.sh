#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${1:-$PWD}"
DIST_DIR="${2:-$PWD/dist}"

cd "$ROOT_DIR"

mkdir -p "pool/main/g/ghnote" \
         "dists/stable/main/binary-amd64" \
         "dists/stable/main/binary-arm64"

cp -f "$DIST_DIR"/*.deb "pool/main/g/ghnote/"

# Regenerate package indexes
rm -f dists/stable/main/binary-amd64/Packages dists/stable/main/binary-amd64/Packages.gz
rm -f dists/stable/main/binary-arm64/Packages dists/stable/main/binary-arm64/Packages.gz

if ls pool/main/g/ghnote/*_amd64.deb >/dev/null 2>&1; then
  dpkg-scanpackages -a amd64 pool > dists/stable/main/binary-amd64/Packages
  gzip -k -f dists/stable/main/binary-amd64/Packages
else
  : > dists/stable/main/binary-amd64/Packages
  gzip -k -f dists/stable/main/binary-amd64/Packages
fi

if ls pool/main/g/ghnote/*_arm64.deb >/dev/null 2>&1; then
  dpkg-scanpackages -a arm64 pool > dists/stable/main/binary-arm64/Packages
  gzip -k -f dists/stable/main/binary-arm64/Packages
else
  : > dists/stable/main/binary-arm64/Packages
  gzip -k -f dists/stable/main/binary-arm64/Packages
fi

apt-ftparchive release dists/stable > dists/stable/Release
