# Alienware Control

An Omarchy 4 shell plugin for fans, lighting and power on the Alienware x15 R2.

Two parts: a Quickshell plugin that runs unprivileged, and `alienwarectl`, a Go binary that is both
a root D-Bus daemon and a user-side CLI.

- Fan speeds, temperatures, GPU draw and power limits in the bar
- Thermal profiles, CPU turbo, RAPL limits and an additive fan curve
- Chassis RGB, four regions, and per key keyboard lighting
- Follows the Omarchy theme, shows mute state on the mute keys, and restores itself after a suspend

Built for and tested on an Alienware x15 R2. Other models in the family differ, in particular the
lighting zone ids and the keyboard index map.

## Install

The plugin and the helper install separately.

```
omarchy plugin add https://github.com/chr0nzz/omarchy-alienware.git --enable

cd ~/.config/omarchy/plugins/xyzlab.alienware/packaging
makepkg -si
sudo systemctl enable --now alienwarectl.service
```

The plugin must be a real directory. The shell's inotify watcher does not follow symlinks, which is
why the package does not ship the plugin itself.

## Update

```
omarchy plugin update xyzlab.alienware

cd ~/.config/omarchy/plugins/xyzlab.alienware/packaging
makepkg -si
sudo systemctl restart alienwarectl.service
```

The QML side hot-reloads on save, so a plugin update needs nothing more than `omarchy restart shell`.
The helper does not: `makepkg` builds the git **tag** named by `pkgver`, never your working tree, so
a helper change is only live once a new tag exists and you have rebuilt. Check what is actually
running with `alienwarectl version`.

## Uninstall

Blank the lighting first. Removing the package does not turn the leds off, and afterwards there is
nothing left to turn them off with.

```
alienwarectl kbd off
alienwarectl rgb off

sudo systemctl disable --now alienwarectl.service
sudo pacman -Rns omarchy-alienware omarchy-alienware-debug

omarchy plugin remove xyzlab.alienware
```

The udev rule keeps applying until a reboot, or `sudo udevadm control --reload && sudo udevadm
trigger`. Your saved colours and settings stay behind in
`~/.local/state/omarchy/alienware/state.json`.

## Keybind

Add to `~/.config/hypr/bindings.lua`:

```lua
o.bind("SUPER SHIFT", "A", "Alienware", "omarchy-shell shell summon xyzlab.alienware '{}'")
```

## Docs

| | |
|---|---|
| [Roadmap](docs/roadmap.md) | What works, what is planned, what this hardware cannot do |
| [Hardware](docs/hardware.md) | The two lighting controllers, the region map, the privilege model |
| [Behaviour](docs/behaviour.md) | Suspend and resume, mute indicator, thermal profiles, G-Mode |
| [CLI](docs/cli.md) | Every `alienwarectl` verb |
| [Key map](KEYMAP.txt) | The measured keyboard index map |
| [Development](docs/development.md) | Tests and releases |
