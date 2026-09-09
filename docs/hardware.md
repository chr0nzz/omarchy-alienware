# Hardware

What the two lighting controllers are, how they are addressed, and why the keyboard stays root only.

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

The full table lives in [`KEYMAP.txt`](KEYMAP.txt), which is the source `Model.js` is derived from.
A test parses both and fails if they disagree, so the two cannot drift apart. To verify or extend
the map, run `alienwarectl kbd identify <idx>`, look at the keyboard, and record what lit.

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
