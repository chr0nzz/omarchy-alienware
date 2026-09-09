# Alienware Control

An Omarchy 4 shell plugin for fans, lighting and power on the Alienware x15 R2.

Plugin id `xyzlab.alienware`. Two parts: a Quickshell plugin that runs unprivileged, and
`alienwarectl`, a Go binary that is both a root D-Bus daemon and a user-side CLI.

## Roadmap

Everything below was established on the author's Alienware x15 R2. Other models in the family
differ, in particular the chassis zone ids and the keyboard index map.

### Working

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
| Chassis RGB | AlienFX ELC over `/dev/hidraw0` | Four regions, unprivileged |
| Keyboard RGB | Darfon AlienFX APIv5 over `/dev/hidraw1` | 88 leds across 84 keys, via the daemon |
| Theme sync | shell accent | Paints chassis and keyboard |
| Mute indicator | PipeWire | The two mute keys follow sink and source mute |
| Restore on resume | logind `PrepareForSleep` | Lighting, and the profile from a pre sleep snapshot |

### Planned

| Item | Where it stands |
|---|---|
| `kbd brightness` | `kbd.Device.SetBrightness` exists in the device layer, no CLI verb exposes it |
| Mute indicator settings | Always on and red. No toggle, no colour picker, not persisted to state |
| RGB effects | `rgb mode` is a recognised verb that always answers `not-supported`. Only flat colours per region are wired up |
| Keyboard effects | Not started. The controller drives its own loop frame, so this is a protocol question, not a UI one |
| Per led addressing inside a ring half | The AW-ELC supports it, the CLI only exposes whole regions |

### Not possible on this hardware

| Item | Why |
|---|---|
| Fan curves that make fans quieter | The kernel exposes one additive `fanN_boost` per fan, not a curve upload. A curve can only raise fans above the firmware's own choice, never lower them. The curve editor draws that floor |
| Battery charge limit | Absent on this model |
| GPU MUX and dynamic boost | Absent on this model |
| Mute and caps lock indicator lamps | Firmware driven off HID state. They are not in the AlienFX matrix and cannot be written |

## RGB regions

The AW-ELC chassis controller (`187c:0550`) exposes four independently addressable regions,
mapped from the real, sparse hardware zone ids:

| Region id | Name | LEDs | Hardware zone ids |
| --- | --- | --- | --- |
| `power` | Power button | 1 | 0 |
| `logo` | Lid logo | 1 | 1 |
| `ring-top` | Ring top | 8 | 8-15 |
| `ring-bottom` | Ring bottom | 8 | 16-23 |

## Keyboard

The keyboard is a separate Darfon controller (`0d62:babc`), speaking AlienFX APIv5 over
`/dev/hidraw1` as 64 byte HID feature reports on feature id `0xcc`. The wire index space is 0 to
135, but most of it is holes: this layout has **88 leds across 84 keys**. See the key map below.

This node has no udev rule and stays root-only, on purpose: it is the same hidraw node the kernel
uses as the keyboard's input interface, so granting the logged-in session an ACL on it would let
any process running as that user read keystrokes. All keyboard writes go through the root daemon
over D-Bus, gated by the `org.xyzlab.alienware.set-keyboard` polkit action, the same pattern as the
fan and profile controls.

Keyboard authorisation is deliberately NON interactive. `CheckAuthorization` is called with no
user interaction flag for `set-keyboard` only, so when the session is locked, such as during the
restore that runs on resume, polkit returns not-authorized instead of raising a password prompt.
The plugin re-arms its keyboard restore on a denied write and the existing retry ladder applies the
colours once the session is unlocked and active again. Lighting must never interrupt the user for a
password, it is not an action they asked for.

| Verb | Effect |
| --- | --- |
| `kbd status` | Report presence and key count, `present:false` cleanly if the controller is absent |
| `kbd set-map <idx=RRGGBB,...>` | Light named keys in one transaction |
| `kbd set-all <RRGGBB>` | Light every key |
| `kbd identify <idx>` | Light exactly one wire index white and blank the rest |
| `kbd off` | Blank every key |

`kbd identify` exists because the index map cannot be reasoned about, only measured. Use it to
verify or extend the map, one index at a time, and trust what lights over what looks obvious.

Every write, whatever its shape, is applied as a single open-device transaction. This controller
family wedges under write pressure, so the daemon holds the device open for its lifetime and
reopens only after an error, rather than opening and closing per call. It also serialises keyboard
calls against each other, and `hidraw.Open` takes an advisory `flock` so the CLI, the daemon and the
plugin cannot hold the same node at once.

## Key map

The wire order follows the keyboard matrix, not the visual layout, and it is full of holes. It was
established by lighting single indices and observing the machine. Do not infer it.

| indices | keys |
| --- | --- |
| 0-15 | esc, f1 to f12, home, end, del |
| 16-19 | volmute, voldown, volup, micmute |
| 20-32 | grave, 1 to 0, minus, equals, contiguous |
| 34-135 | the rest, sparsely, see the table in the source |

The media keys sit BELOW the number row, not spliced into the middle of it, and the number row is
contiguous.

**Four wide keys carry two leds each.** Painting only the primary lights half the key:

| key | indices |
| --- | --- |
| backspace | 34 + 35 |
| caps | 60 + 61 |
| lshift | 80 + 81 |
| lsuper | 102 + 103 |

Every other wide key, tab, backslash, enter, rshift, ctrl and alt, is a single led. `kbKey` carries
an `indices` array and `cmdKbdSetMap` emits every led for a key, so one paint covers the whole key.

The space bar has no led. The remaining 48 indices are matrix holes. There is no addressable zone
for the mute indicators and the caps lock dot is not in the matrix either: both are firmware driven
off HID state and cannot be written over AlienFX.

## Mute indicator

The two mute keys follow PipeWire. `pactl subscribe` provides the events, `wpctl get-volume` reads
the state, and a muted sink paints `volmute` (index 16) while a muted source paints `micmute`
(index 19) red.

It is an overlay, applied at write time on top of the effective key map and never stored into the
saved colours, so muting cannot eat whatever colour you picked for those two keys. The indicator
shows even when that key is individually switched off, because an indicator you have turned off is
useless, but it stays dark when the master lights toggle is off. Reads are debounced by 400ms, since
this controller wedges when transactions arrive too close together.

## Privilege

Sysfs control nodes are root-owned, so fan, thermal, turbo and power writes go through the daemon
on the D-Bus system bus, gated by polkit. The plugin reads sysfs directly because it is
world-readable.

RGB is outside that boundary. A udev rule tags the AW-ELC HID node `uaccess`, which gives the
logged-in session an ACL on it, so lighting is driven directly over `/dev/hidraw0` unprivileged
and never touches the daemon. No setuid binaries, no root shell. Check it before reporting an RGB
fault:

```
getfacl /dev/hidraw0 | grep "$USER"
```

If the logged-in user has no entry there, the udev rule is missing or does not match this
controller, and every RGB command will fail.

**The AW-ELC controller (187c:0550) wedges if two processes open its HID node at once.** Every
`hidraw.Open` now takes an advisory `flock`, so two writers cannot collide and a genuinely
contended node reports `hidraw-busy` after a bounded wait instead of wedging. The historical
symptom is a status report that reads back all zero. `alienwarectl` detects that on session open,
USB resets the controller once, reconnects and retries, without root. If the controller still
reports an all-zero status after that, `alienwarectl rgb reset` is a manual escape hatch that does
the same reset on its own. There is a second, separate failure mode where the controller accepts
writes but the LEDs stay frozen on an old frame. That one cannot be detected from a status read, so
`rgb reset` is the fix for it too, run by hand.

## Resume from suspend

Both controllers drop their LED frames across a suspend. Service.qml watches the logind
PrepareForSleep signal over dbus-monitor and, on the resume edge, calls reload(), which re-arms the
restore latches and re-applies the saved chassis and keyboard colours. The existing retry ladders
cover the window where the USB devices have not finished re-enumerating yet.

The thermal profile is restored the same way, but by snapshot rather than by preference. On the
pre-sleep edge the plugin records whatever profile sysfs currently reports. On resume it waits for a
status poll newer than the resume, and if the profile has drifted it writes the snapshot back, then
keeps checking every 2 seconds for 8 tries so it lands after power-profiles-daemon rather than
before it. It stores no desired profile and asserts nothing on a cold start, so changing the profile
with powerprofilesctl or the Omarchy menu is never fought.

## power-profiles-daemon

power-profiles-daemon exposes three of the six profiles and re-applies its own choice on resume
and on AC changes.

This plugin treats sysfs as authoritative and leaves the daemon running. It restores the profile
across a suspend by snapshotting it beforehand, as described above, and gives up after 8 tries
rather than fighting forever. If profiles still revert, mask it:

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

**The PKGBUILD builds a TAG, not your working tree.** `source` is
`git+${url}.git#tag=v${pkgver}`, so running `makepkg -si` with a stale `pkgver` will happily
DOWNGRADE the installed package to whatever that tag holds. The two halves of this plugin therefore
deploy differently:

- QML and JS hot-reload on save. `omarchy restart shell` is enough.
- Anything under `helper/` needs a version bump, a pushed `vX.Y.Z` tag, then `makepkg -si` and
  `sudo systemctl restart alienwarectl.service`.

Check what is actually running with `alienwarectl version` rather than assuming a rebuild took.

The plugin must be a real directory. The shell's inotify watcher does not follow symlinks, which
is why the package does not ship the plugin itself.

## Uninstall

Blank the lighting FIRST. Removing the package does not turn the leds off. Both controllers hold
whatever frame was last written to them, so the keyboard and the chassis stay lit at their current
colours until something else writes to them or the machine power cycles. Afterwards the binary is
gone and the uaccess ACL with it, so there is nothing left to blank them with.

```
alienwarectl kbd off
alienwarectl rgb off
```

Then the helper:

```
sudo systemctl disable --now alienwarectl.service
sudo pacman -Rns omarchy-alienware omarchy-alienware-debug
```

That removes the binary, the systemd unit, the D-Bus service and policy, the polkit action and
rules, and the `71-alienware-aw-elc.rules` udev rule. The package declares no backup files, so
nothing is left behind as `.pacsave`. The udev rule keeps applying to the already enumerated device
until a reboot, or:

```
sudo udevadm control --reload && sudo udevadm trigger
```

Then the plugin:

```
omarchy plugin remove xyzlab.alienware
```

If you would rather do it by hand, delete `~/.config/omarchy/plugins/xyzlab.alienware`, drop the
widget from `~/.config/omarchy/shell.json`, and run `omarchy restart shell`.

Two things neither step removes, both outside any package:

| path | what it is |
| --- | --- |
| `~/.local/state/omarchy/alienware/state.json` | saved colours, fan curves and settings |
| `~/.cache/omarchy-alienware/` | including `KEYMAP.txt` |

`KEYMAP.txt` is the measured wire index map. It was established by lighting single indices and
looking at the machine, it cannot be derived from the layout, and it is worth keeping even if the
plugin goes.

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
alienwarectl kbd status
alienwarectl kbd set-map 0=ff0000,4=00ff00
alienwarectl kbd set-all ff8800
alienwarectl kbd off
```

`rgb set`, `rgb identify` and the `region=colour` pairs in `rgb set-map` take a region id from the
[RGB regions](#rgb-regions) table: `power`, `logo`, `ring-top` or `ring-bottom`. An unknown region
id is a `bad-request` error that names the valid ones. `rgb mode` is recognised but always answers
`not-supported`, lighting effects are not implemented yet.

`kbd` verbs are daemon clients like `profile`, `boost` and `pl`: they never open the keyboard's
hidraw node directly, only the root daemon does. `idx` in `kbd set-map` is the raw wire key index,
0-135.

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
