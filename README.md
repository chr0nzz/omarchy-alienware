# Alienware Control

An Omarchy 4 shell plugin for fans, lighting and power on the Alienware x15 R2.

Two parts: a Quickshell plugin that runs unprivileged, and `alienwarectl`, a Go binary that is both
a root D-Bus daemon and a user-side CLI.

- Fan speeds, temperatures, GPU draw and power limits in the bar
- Thermal profiles, CPU turbo, RAPL limits and an additive fan curve
- Chassis RGB, four regions, and per key keyboard lighting
- Follows the Omarchy theme, shows mute state on the mute keys, and restores itself after a suspend

Tested on an Alienware x15 R2. Other models in the family differ, in particular the lighting zone
ids and the keyboard index map.

## Install

The plugin and the helper install separately.

```
omarchy plugin add https://github.com/chr0nzz/omarchy-alienware.git --enable

cd ~/.config/omarchy/plugins/xyzlab.alienware/packaging/bin
makepkg -si
sudo systemctl enable --now alienwarectl.service
```

That installs the prebuilt binary from the latest release. It needs no Go toolchain and takes
seconds. To build from source instead, run `makepkg -si` in `packaging/` rather than
`packaging/bin/`. Both install the same eight files and either can be removed with `pacman -Rns`.

The plugin must be a real directory. The shell's inotify watcher does not follow symlinks, which is
why the package does not ship the plugin itself.

## Update

```
omarchy plugin update xyzlab.alienware

cd ~/.config/omarchy/plugins/xyzlab.alienware/packaging/bin
makepkg -si
sudo systemctl restart alienwarectl.service
```

The QML side hot-reloads on save, so a plugin update needs nothing more than `omarchy restart shell`.
The helper does not. Both packages install a published release, never your working tree, so a helper
change is only live once a release exists and you have rebuilt. Check what is actually running with
`alienwarectl version`.

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

## If the panel will not close

It takes exclusive keyboard focus while it is open, so a wedged panel can look like a wedged
machine. It is not. Either of these closes it, and neither needs the keyboard focus back:

```
omarchy-shell alienware close
omarchy-shell shell hide xyzlab.alienware
```

Clicking anywhere outside the panel closes it too, and `esc` closes it unless a text field has
focus, in which case the first `esc` leaves the field and the second closes the panel.

## Docs

| | |
|---|---|
| [Roadmap](docs/roadmap.md) | What works and what is planned |
| [Hardware](docs/hardware.md) | The two lighting controllers, the region map, the privilege model |
| [Behaviour](docs/behaviour.md) | Suspend and resume, mute indicator, thermal profiles, G-Mode |
| [CLI](docs/cli.md) | Every `alienwarectl` verb |
| [Key map](docs/KEYMAP.txt) | The measured keyboard index map |
| [Development](docs/development.md) | Tests and releases |
