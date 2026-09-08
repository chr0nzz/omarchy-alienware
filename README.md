# Alienware Control

An Omarchy 4 shell plugin for fans, lighting and power on the Alienware x15 R2.

Plugin id `xyzlab.alienware`. Two parts: a Quickshell plugin that runs as your user, and
`alienwarectl`, a Go binary that is both a root D-Bus daemon and a user-side CLI.

## What works on this hardware

| Control | Interface | State |
|---|---|---|
| Fan RPM and temps | hwmon `alienware_wmi` | Read |
| Secondary temps | hwmon `dell_smm` | Read |
| GPU power draw | `nvidia-smi` | Read |
| GPU power limit | `nvidia-smi` | Read only, follows the thermal profile |
| Thermal profiles | `platform_profile` | Read and write |
| CPU turbo | `intel_pstate/no_turbo` | Read and write |
| PL1 / PL2 / peak | `intel-rapl:0` | Write if firmware permits |
| Fan boost | `fanN_boost` | Write, additive only |
| Chassis RGB | AlienFX ELC over `/dev/hidraw0` | Write, no root |
| Keyboard RGB | Darfon controller | Not supported yet |
| Battery charge limit | none | Absent on this model |
| GPU MUX / dynamic boost | none | Absent on this model |

## Two things the UI does not pretend about

**Fan curves cannot make fans quieter.** The kernel exposes one additive `fanN_boost` value per
fan, not a curve upload. A curve raises fans above the firmware's own choice and can never lower
them. The curve editor draws that floor.

**RGB lighting effects are not implemented.** `rgb mode` is a recognised verb that always answers
`not-supported`. Only flat colours per region are wired up right now.

## RGB regions

The AW-ELC chassis controller (`187c:0550`) exposes four independently addressable regions,
mapped from the real, sparse hardware zone ids:

| Region id | Name | LEDs | Hardware zone ids |
| --- | --- | --- | --- |
| `power` | Power button | 1 | 0 |
| `logo` | Lid logo | 1 | 1 |
| `ring-top` | Ring top | 8 | 8-15 |
| `ring-bottom` | Ring bottom | 8 | 16-23 |

The keyboard is a separate Darfon controller (`0d62:babc`), root-only with no user ACL. It is not
driven by `alienwarectl` yet and keeps whatever colour it last had.

## Privilege

Sysfs control nodes are root-owned, so fan, thermal, turbo and power writes go through the daemon
on the D-Bus system bus, gated by polkit. The plugin reads sysfs directly because it is
world-readable.

RGB is outside that boundary. A udev rule tags the AW-ELC HID node `uaccess`, which gives your
logged-in session an ACL on it, so lighting is driven directly over `/dev/hidraw0` as you and
never touches the daemon. No setuid binaries, no root shell. Check it before reporting an RGB
fault:

```
getfacl /dev/hidraw0 | grep "$USER"
```

If your user has no entry there, the udev rule is missing or does not match this controller, and
every RGB command will fail.

**The AW-ELC controller (187c:0550) wedges if two processes open its HID node at once.** The
symptom is a status report that reads back all zero. `alienwarectl` detects that on session open,
USB resets the controller once, reconnects and retries, without root. If the controller still
reports an all-zero status after that, `alienwarectl rgb reset` is a manual escape hatch that does
the same reset on its own. There is a second, separate failure mode where the controller accepts
writes but the LEDs stay frozen on an old frame. That one cannot be detected from a status read, so
`rgb reset` is the fix for it too, run by hand.

## power-profiles-daemon

power-profiles-daemon exposes three of the six profiles and re-applies its own choice on resume
and on AC changes.

This plugin treats sysfs as authoritative, leaves the daemon running, and re-asserts your profile
after resume. If you still see profiles reverting, mask it:

```
systemctl mask power-profiles-daemon
```

## G-Mode

`force_gmode` is a module parameter, not a runtime toggle. Without it, `performance` is the
strongest profile. With it, `performance` becomes G-Mode:

```
echo "options alienware_wmi force_gmode=1" | sudo tee /etc/modprobe.d/alienware.conf
```

Reboot to apply. The panel labels the button according to what it reads back.

## Install

The bar plugin and the helper install separately.

Plugin, cloned by Omarchy into `~/.config/omarchy/plugins/xyzlab.alienware`:

```
omarchy plugin add https://github.com/chr0nzz/omarchy-alienware.git --enable
```

Helper, from the cloned repo:

```
cd ~/.config/omarchy/plugins/xyzlab.alienware/packaging && makepkg -si
sudo systemctl enable --now alienwarectl.service
```

The plugin must be a real directory. The shell's inotify watcher does not follow symlinks, which
is why the package does not ship the plugin itself.

## Keybind

Omarchy 4 uses a Lua Hyprland config, and `hyprctl keyword` is rejected by the non-legacy parser.
Add the bind to `~/.config/hypr/bindings.lua`:

```lua
o.bind("SUPER + A", "Alienware Control", "omarchy-shell alienware open")
```

## CLI

```
alienwarectl status
alienwarectl profile balanced
alienwarectl boost cpu 120
alienwarectl curve apply curve.json
alienwarectl curve stop
alienwarectl turbo on
alienwarectl pl 1 65
alienwarectl rgb status
alienwarectl rgb set logo ff0044
alienwarectl rgb set-all ff8800
alienwarectl rgb set-map logo=ff0000,power=00ff00
alienwarectl rgb brightness 75
alienwarectl rgb identify ring-top
alienwarectl rgb off
alienwarectl rgb reset
```

`rgb set`, `rgb identify` and the `region=colour` pairs in `rgb set-map` take a region id from the
[RGB regions](#rgb-regions) table: `power`, `logo`, `ring-top` or `ring-bottom`. An unknown region
id is a `bad-request` error that names the valid ones. `rgb mode` is recognised but always answers
`not-supported`, lighting effects are not implemented yet.

Every command prints JSON and exits non-zero on failure.

## Fail-safe

The daemon writes boost 0 on start, when a curve stops, on SIGTERM and SIGINT, and from
`ExecStopPost` so a crash or SIGKILL cannot leave the fans pinned. A systemd watchdog restarts a
hung control loop.

## Releases

Tagging `v*` builds a static `alienwarectl`, verifies it, and publishes it with checksums. Grab it
instead of building:

```
gh release download --repo chr0nzz/omarchy-alienware --pattern 'alienwarectl-*-x86_64.tar.gz'
```

`makepkg` still builds from source and is the supported path.

## Tests

```
npm test
cd helper && go test ./...
```

## Status

Built against verified interface readings from the target machine. The QML has not been run, and
no code here has been exercised against the Alienware hardware itself.
