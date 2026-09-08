# alienfx-spike

A pure-Go, no-cgo spike CLI for talking to the Alienware AW-ELC lighting
controller (187c:0550, "API_V4" in the T-Troll AlienFX-SDK) directly over
`/dev/hidrawN`. This exists to discover the real, sparse hardware zone IDs
on an Alienware x15 R2 so the shipping `omarchy-alienware` plugin can
eventually stop depending on OpenRGB for RGB. It is exploratory: nothing
here is wired into `helper/` and nothing here should be treated as
production code.

## Read this before touching hardware

The AW-ELC controller's firmware wedges if two processes open its HID node
at the same time. `openrgb-server` holds `/dev/hidrawN` open whenever it is
running.

- Stop `openrgb-server` before running any `afxspike` command that talks to
  the device (`probe`, `sweep`, `set`, `all`, `off`). `version` and `help`
  never open the device and are always safe.
- `probe` reports `looksWedged: true` when it can read a status report but
  every byte in it is zero, one of the symptoms you already described for a
  stuck controller. Treat that as "stop, power-cycle, and check for a second
  process," not as "keep sending commands."
- This spike was built and unit-tested only. It has not been run against
  `/dev/hidraw0` or any other real device by the agent that wrote it. All
  hardware verification is on you.

## Build and test

```bash
cd spike
go build ./...
gofmt -l .
go vet ./...
go test ./...
```

All of the above are clean as of this writing. `CGO_ENABLED=0 go build ./...`
also succeeds; the module has no C dependency and no non-stdlib imports.

## CLI usage

```
afxspike <verb> [args] [--device=/dev/hidrawN] [--pid=0550] [--transport=output|write|feature]
```

| Verb | Example | Does |
| --- | --- | --- |
| `probe` | `afxspike probe` | Opens the device, reads status and raw config, prints JSON, flags a likely wedge |
| `sweep <first> <last>` | `afxspike sweep 0 19 --dwell=2s` | Lights zone IDs one at a time across the range, white, printing each as it goes |
| `set <zoneid> <RRGGBB>` | `afxspike set 4 ff8800` | Lights a single hardware zone ID |
| `all <RRGGBB>` | `afxspike all 00ff88` | Lights a default sweep of zone IDs (0-31) |
| `off` | `afxspike off` | Blanks the same default sweep of zone IDs (0-31) |
| `version` | `afxspike version` | Prints the CLI version, never touches the device |

Every verb prints one JSON object per line to stdout and exits non-zero on
failure, with `{"ok":false,"error":"...","code":"bad-request"|"internal"}`.
`sweep` prints one success line per zone id as it drives it, in addition to
the usual exit code.

Flags, any position after the verb:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--device=/dev/hidrawN` | auto-located by VID/PID | Skip discovery and use this node directly |
| `--pid=0550` | `0550` | Override the product id used for auto-discovery (vendor is fixed at `187c`) |
| `--transport=output\|write\|feature` | `output` | Which HID report type carries outgoing commands, see "Biggest open question" below |
| `--dwell=2s` (`sweep` only) | `2s` | How long each zone stays lit before advancing |

## File layout

| Path | Role |
| --- | --- |
| `cmd/afxspike/main.go` | Entry point |
| `internal/hidraw` | Pure-Go hidraw layer: ioctl request-number computation, feature/input/output report get/set, VID/PID node discovery |
| `internal/elc` | AlienFX ELC wire encoding and the `Device` API built on top of it |
| `internal/cli` | Verb dispatch and JSON output |

## What was ported, and where from

Everything below is read from `AlienFX-SDK/src/AlienFX_SDK.cpp` and
`AlienFX-SDK/include/alienfx_control.h` in the reference clone, specifically
the `API_V4` ("common tron/desktop") branches, which is what this exact
device (187c:0550, 34-byte HID report length) maps to in
`Functions::AlienFXProbeDevice`.

| `internal/elc` | Reference source | Opcode |
| --- | --- | --- |
| `ControlFrame` | `COMMV4_control`, used by `Reset()` and `UpdateColors()` | 0x21 |
| `SelectZonesFrame` | `COMMV4_colorSel`, used by `SetV4Action` | 0x23 |
| `AddActionFrame` | `COMMV4_colorSet`, used by `SetV4Action` | 0x24 |
| `SetOneColorFrame` | `COMMV4_setOneColor`, used by `SetAction`'s plain-color path and `SetMultiColor` | 0x27 |
| `TurnOnFrame` | `COMMV4_turnOn`, used by `SetBrightness` | 0x26 |
| `SetPowerControlFrame` | `COMMV4_setPower`, used by `SetPowerAction` (ported, not exercised by the CLI) | 0x22 |

`Device.Reset()` reproduces `Functions::Reset()` for API_V4 exactly:
wait for ready, send control(type=4, "remove"), send control(type=1, "start
new"). `Device.Apply()` reproduces `Functions::UpdateColors()`: send
control(type=3, "finish and play") with no field overrides. `Device.Status()`
reproduces `Functions::GetDeviceStatus()`: the status byte lives at offset 2
of the report, and the five named values (`StatusV4Ready`=33, `Busy`=34,
`WaitColor`=35, `WaitUpdate`=36, `WasOn`=38) are the reference header's
`ALIENFX_V4_*` constants, copied as decimal, not reinterpreted as hex.

Every frame builder in `internal/elc/protocol.go` was checked field-by-field
against the C++ source rather than against the header's comments, which
occasionally disagree with the code that ships (for example, the byte
offsets `COMMV4_colorSel`'s comment claims versus what `SetV4Action`
actually writes). `protocol_test.go` encodes each byte offset from the code
path, not the comment.

The ioctl request numbers for `HIDIOCSFEATURE`, `HIDIOCGFEATURE`,
`HIDIOCSINPUT`, `HIDIOCGINPUT`, `HIDIOCSOUTPUT`, `HIDIOCGOUTPUT`, and
`HIDIOCGRAWINFO` were computed from `<asm-generic/ioctl.h>`'s `_IOC` macro
and cross-checked by compiling a small C program against this machine's own
`/usr/include/linux/hidraw.h`; the printed values match `hidraw_test.go`'s
golden constants exactly. hidraw node discovery by VID/PID was checked
against this machine's real `/sys/class/hidraw/hidraw0/device/uevent`,
which reports `HID_ID=0003:0000187C:00000550` for the AW-ELC controller.

## The biggest open question: which HID report type actually works

The reference SDK's API_V4 branch sends every write as an **Output** report
(`HidD_SetOutputReport` → `hid_send_output_report`) and reads status as an
**Input** report (`HidD_GetInputReport` → `hid_get_input_report`). Neither
is a Feature report. Only the API_V5 and API_V8 branches in the same file
use `hid_send_feature_report` / `hid_get_feature_report`.

That is a direct conflict with this spike's brief, which was written from
your OpenRGB testing and named Feature reports specifically. Two things can
both be true here: the T-Troll SDK is authoritative for the byte layout
(command opcodes, field offsets), while OpenRGB's own, separately-written
Alienware controller might use a different HID report type than T-Troll's
SDK to reach the same device, especially since T-Troll's SDK goes through
libusb directly rather than through the Linux hidraw character device, and
those two paths don't necessarily pick the same USB HID report type for a
"write bytes to the device" operation. I don't have OpenRGB's
`AlienwareController.cpp` on this machine to settle it either way.

Rather than guess, `internal/elc` exposes all three plausible transports
behind `--transport`:

| `--transport` | Mechanism | Matches |
| --- | --- | --- |
| `output` (default) | `HIDIOCSOUTPUT` ioctl | The literal kernel equivalent of a libusb `SET_REPORT(Output)` control transfer, i.e. what `hid_send_output_report` does under hidapi's libusb backend, which is what the reference SDK actually links against |
| `write` | plain `write(2)` to the hidraw node | The idiomatic hidraw way to send an Output report; the kernel driver falls back to a control-transfer `SET_REPORT` itself if the device has no interrupt OUT endpoint, which this controller likely doesn't |
| `feature` | `HIDIOCSFEATURE` ioctl | This spike's original brief, and possibly what OpenRGB does |

`probe` doesn't have to guess for reads: it always attempts both an Input
report (`HIDIOCGINPUT`, matching the reference's `GetDeviceStatus` exactly)
and a Feature report (`HIDIOCGFEATURE`) and prints both results, so one
`probe` run tells you which one returns a real, non-zero status rather than
an ioctl error or all-zero bytes.

**When you get to hardware**: run `probe` first with openrgb-server stopped
and check which read path returns sane data. Then try `set <a-known-lit-id>
<color>` with `--transport=output` first (it's the default and the most
faithful port of the reference), and fall back to `--transport=feature` if
nothing lights. Whichever one works should get promoted to the default in
any follow-up, and the other two are worth deleting once that's settled so
this doesn't ship three untested code paths.

## Other places I guessed, diverged, or left gaps

- **No documented "read platform id" command.** The reference SDK never
  queries the device for a platform id or zone table; `AlienFXProbeDevice`
  decides the device is API_V4 purely from VID/PID and the USB HID report
  length (33/34 bytes), no round trip to the hardware. `probe`'s status and
  feature reads are the closest available substitute, not a documented
  "give me your config" command. The `0x306` platform id you saw from
  OpenRGB must come from OpenRGB's own detection code, which isn't in the
  reference clone.
- **Sparse zone IDs are still unverified for x15 R2.** Nothing in the
  reference SDK or this port knows the real zone list for this exact chip.
  `sweep` exists specifically to find it. `all` and `off` default to a
  1-by-1 sweep of IDs 0-31 as a plausible superset, not the real sparse set
  T-Troll published for the sibling 0x551 chip (0, 1, 2, 4). Once `sweep`
  finds the real IDs, `all`/`off` should switch to that exact list.
- **`WaitForReady` is bounded, the reference isn't.** `Functions::Reset()`
  for API_V4 calls `while (!IsDeviceReady()) usleep(20);` with no timeout.
  A CLI spike against hardware that's known to wedge shouldn't be able to
  hang forever, so `Device.WaitForReady` takes a 2 second timeout and
  returns `ErrDeviceTimeout` instead. Everything else about the ready/busy
  state machine is a direct port.
- **Brightness is simplified, not exposed.** The reference's
  `SetBrightness` scales by a per-light global-brightness byte and a
  device-specific scale table meant for multi-device setups; this spike
  doesn't track any of that, so `Device.Dim` takes a plain 0-100 percent
  and sends `100-percent` as the dim byte. It's ported and unit-tested but
  not wired to a CLI verb, since brightness wasn't in the required verb
  list.
- **Power-button state (`SetPowerAction`, `SaveLightsState`,
  `SaveLightsStateToStartup`) was not ported.** That's a separate, larger
  state machine in the reference for writing startup/AC/battery defaults
  into the device's persistent memory across zone IDs 0x5b-0x60. None of
  the required verbs touch it, and persistent-memory writes seemed like
  the wrong thing to add speculatively to a spike against hardware that
  wedges.
- **hidraw device discovery picks the first match.** If more than one node
  ever matches VID 187c and the given PID (shouldn't happen for this
  device on a normal system), `FindOneByVIDPID` takes the lexicographically
  first `/dev/hidrawN` path. Use `--device=` to force a specific node.
