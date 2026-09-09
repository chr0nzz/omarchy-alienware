#!/usr/bin/env bash
# Point the prebuilt package at a published release.
#   ./update.sh 0.3.3
set -euo pipefail
version="${1:?usage: update.sh <version, without the v>}"
repo=chr0nzz/omarchy-alienware
here="$(cd "$(dirname "$0")" && pwd)"

sums=$(gh release download "v${version}" --repo "$repo" --pattern sha256sums.txt --output - 2>/dev/null) \
  || { echo "no published release v${version}, or its assets are missing" >&2; exit 1; }

sha=$(awk -v f="alienwarectl-${version}-x86_64.tar.gz" '$2 == f {print $1}' <<<"$sums")
[ -n "$sha" ] || { echo "sha256sums.txt has no entry for alienwarectl-${version}-x86_64.tar.gz" >&2; exit 1; }

sed -i "s/^pkgver=.*/pkgver=${version}/" "$here/PKGBUILD"
sed -i "s/^sha256sums=.*/sha256sums=('${sha}')/" "$here/PKGBUILD"

echo "pkgver=${version}"
echo "sha256sums=('${sha}')"
