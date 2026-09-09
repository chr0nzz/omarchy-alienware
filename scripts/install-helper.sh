#!/usr/bin/env bash
set -euo pipefail

here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
pkgdir="${here}/packaging/bin"

pause() {
  if [ -t 0 ]; then
    printf '\nPress enter to close this window '
    read -r _ || true
  fi
}
trap pause EXIT

echo "Alienware helper install"
echo
echo "This runs:"
echo "  cd ${pkgdir}"
echo "  makepkg -si"
echo "  sudo systemctl enable --now alienwarectl.service"
echo
echo "makepkg builds a package from the published release."
echo "pacman and systemctl will ask for your password."
echo

if [ ! -d "${pkgdir}" ]; then
  echo "Cannot find ${pkgdir}"
  echo "Install the plugin first:"
  echo "  omarchy plugin add https://github.com/chr0nzz/omarchy-alienware.git --enable"
  exit 1
fi

if [ -t 0 ]; then
  printf 'Press enter to continue, or ctrl-c to stop '
  read -r _
fi

cd "${pkgdir}"
makepkg -si
sudo systemctl enable --now alienwarectl.service

echo
alienwarectl version
echo
echo "Done. The panel picks the helper up within a few seconds."
