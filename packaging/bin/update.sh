#!/usr/bin/env bash
set -euo pipefail

version="${1:?usage: update.sh <version, without the v>}"
repo=chr0nzz/omarchy-alienware
url="https://github.com/${repo}"
here="$(cd "$(dirname "$0")" && pwd)"
src="$(cd "${here}/.." && pwd)"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

sums=$(gh release download "v${version}" --repo "$repo" --pattern sha256sums.txt --output - 2>/dev/null) \
  || { echo "no published release v${version}, or its assets are missing" >&2; exit 1; }

asset="alienwarectl-${version}-x86_64.tar.gz"
binsha=$(awk -v f="$asset" '$2 == f {print $1}' <<<"$sums")
[ -n "$binsha" ] || { echo "sha256sums.txt has no entry for ${asset}" >&2; exit 1; }

commit=$(gh api "repos/${repo}/git/ref/tags/v${version}" --jq '.object.sha' 2>/dev/null) \
  || { echo "cannot resolve the commit behind tag v${version}" >&2; exit 1; }

kind=$(gh api "repos/${repo}/git/ref/tags/v${version}" --jq '.object.type' 2>/dev/null)
if [ "$kind" = "tag" ]; then
  commit=$(gh api "repos/${repo}/git/tags/${commit}" --jq '.object.sha' 2>/dev/null) \
    || { echo "cannot dereference annotated tag v${version}" >&2; exit 1; }
fi

case "$commit" in
  [0-9a-f]*) ;;
  *) echo "resolved commit ${commit} is not a hex sha" >&2; exit 1 ;;
esac
[ "${#commit}" -eq 40 ] || { echo "resolved commit ${commit} is not 40 characters" >&2; exit 1; }

curl -fsSL "${url}/archive/${commit}.tar.gz" -o "${tmp}/src.tar.gz" \
  || { echo "cannot download the source archive for ${commit}" >&2; exit 1; }

srcsha=$(sha256sum "${tmp}/src.tar.gz" | cut -d' ' -f1)
[ -n "$srcsha" ] || { echo "cannot hash the source archive" >&2; exit 1; }

tar -tzf "${tmp}/src.tar.gz" | head -1 | grep -qx "omarchy-alienware-${commit}/" \
  || { echo "the source archive does not unpack to omarchy-alienware-${commit}/" >&2; exit 1; }

sed -i "s/^pkgver=.*/pkgver=${version}/" "${here}/PKGBUILD"
sed -i "s/^sha256sums=.*/sha256sums=('${binsha}')/" "${here}/PKGBUILD"

sed -i "s/^pkgver=.*/pkgver=${version}/" "${src}/PKGBUILD"
sed -i "s/^_commit=.*/_commit=${commit}/" "${src}/PKGBUILD"
sed -i "s/^sha256sums=.*/sha256sums=('${srcsha}')/" "${src}/PKGBUILD"

grep -qx "_commit=${commit}" "${src}/PKGBUILD" || { echo "failed to pin _commit" >&2; exit 1; }
grep -qx "sha256sums=('${srcsha}')" "${src}/PKGBUILD" || { echo "failed to pin the source digest" >&2; exit 1; }
grep -qx "sha256sums=('${binsha}')" "${here}/PKGBUILD" || { echo "failed to pin the binary digest" >&2; exit 1; }

echo "source  pkgver=${version} _commit=${commit}"
echo "source  sha256sums=('${srcsha}')"
echo "bin     pkgver=${version}"
echo "bin     sha256sums=('${binsha}')"
