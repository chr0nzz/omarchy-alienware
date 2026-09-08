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

## kbdspike: the per-key keyboard controller (0d62:babc, APIv5)

A second, separate spike CLI, `kbdspike`, targets a different chip on this
machine: the Darfon per-key RGB keyboard controller at USB 0d62:babc, exposed
as `/dev/hidraw1`. This is the keyboard's own input node. It is root-only by
design (granting user access would expose keystrokes to any process), so
`kbdspike` must run as root, through the project's root D-Bus daemon in the
long run. It shares nothing with `afxspike`/`elc` beyond the pure-Go
`internal/hidraw` layer: different device, different report format, different
wire protocol (T-Troll's SDK calls this "API_V5").

```bash
cd spike
go build ./...
gofmt -l .
go vet ./...
go test ./...
```

### CLI usage

```
kbdspike <verb> [args] [--device=/dev/hidrawN]

  probe                     open the device and print protocol and status info as JSON
  all <RRGGBB>              set every key to one colour
  key <index> <RRGGBB>      set a single key by its protocol index
  sweep <first> <last>      light key indices one at a time, flags: --dwell=2s
  off                       set every key to black
```

Every verb prints one JSON object per line to stdout and exits non-zero on
failure, `{"ok":false,"error":"...","code":"bad-request"|"internal"}`.
`--device` overrides hidraw node discovery, which otherwise locates the node
by VID/PID 0d62:babc.

### What the reference SDK says about APIv5

Read from `AlienFX-SDK/include/alienfx_control.h` (the `COMMV5_*` byte
arrays), `AlienFX-SDK/include/AlienFX_SDK.h` (the `API_V5`/`ALIENFX_V5_*`
constants), and `AlienFX-SDK/src/AlienFX_SDK.cpp` (`Functions::PrepareAndSend`,
`Reset`, `UpdateColors`, `AddV5DataBlock`, `SetAction`, `SetMultiAction`,
`SetMultiColor`, `GetDeviceStatus`, `IsDeviceReady`, `SetBrightness`).

| Question | Answer | Source |
| --- | --- | --- |
| Report size | 64 bytes total, including the report id byte | `Afx_Version` comment `API_V5 = 5, // 64`, cross-checked against this machine's own research notes |
| Report id / feature id | `0xcc`, at byte 0 | `reportIDList[API_V5] == 0xcc` in `alienfx_control.h`, written into `buffer[0]` in `PrepareAndSend` after the raw command bytes are copied in |
| Report type | Feature report, both ways | `PrepareAndSend`'s `case API_V5: result = HidD_SetFeature(...)`, and `GetDeviceStatus`'s `case API_V5: ... HidD_GetFeature(...)`. Never an Output report or plain `write()`, unlike APIv4 |
| Reset opcode | `0x94` | `COMMV5_reset` |
| Status query opcode | `0x93` (write), status byte at offset 2 of the feature-report read that follows | `COMMV5_status`, `GetDeviceStatus` |
| Set-colours opcode | `0x8c 0x02`, then up to 15 four-byte key blocks starting at offset 4 | `COMMV5_colorSet`, `AddV5DataBlock`, `SetMultiAction`'s `bPos` loop (`bPos < length; bPos += 4`) |
| Loop/commit-batch opcode | `0x8c 0x13`, sent once after all colour-set chunks for a given write | `COMMV5_loop`, called outside the chunking loop in `SetMultiAction`/`SetMultiColor` |
| Apply/update opcode | `0x8b 0x01 0xff` | `COMMV5_update`, sent by `Functions::UpdateColors()`, called by application code (`Example-App`) after `SetMultiAction`, not automatically by it |
| Key addressing | One byte per key block, value is **key index + 1**, not the raw index | `AddV5DataBlock`: `{(uint8_t)(index + 1), c->r, c->g, c->b}`, with the reference's own porter noting `// NOTE: +1 because parts start from 1, 0 is for reset? ig` |
| Colour encoding | 8 bits per channel, R, G, B, in that byte order, no brightness byte per key | Same `AddV5DataBlock` line |
| Ready/busy signal | Status byte `!= 0x80` means ready | `IsDeviceReady`, `ALIENFX_V5_WAITUPDATE = 0x80` |

### What was ported faithfully

- `ResetFrame` (`0x94`), `StatusFrame` (`0x93`), `ColorSetFrame` (`0x8c 0x02`
  plus up to 15 `[index+1, R, G, B]` blocks), `LoopFrame` (`0x8c 0x13`),
  `UpdateFrame` (`0x8b 0x01 0xff`), and `TurnOnFrame` (`0x83 0x38 0x9c` plus a
  brightness byte at offset 4) are byte-for-byte reproductions of
  `PrepareAndSend`'s output for the matching `COMMV5_*` command array plus
  the field overrides each caller applies, worked out by hand-simulating
  `memcpy(buffer, command, command[0]+1); buffer[0] = reportIDList[version];`
  for each command.
- The 15-keys-per-frame chunking, and sending exactly one `LoopFrame` after
  all chunks (not one per chunk), reproduces `SetMultiAction`'s `API_V5`
  branch exactly.
- The key-index-plus-one wire encoding is carried over exactly as coded in
  `AddV5DataBlock`, uncertainty comment included below.
- `Device.SetKeyColors` composes reset, chunked colour-set, loop, and update
  into one call, because every real caller in the reference (`Example-App`)
  does exactly that sequence: `SetMultiAction(...)` then `UpdateColors()`.

### What was inferred or intentionally diverged

1. **The `GetDeviceStatus` read is corrected for Linux hidraw semantics.**
   In the C++ source, `HidD_GetFeature(devHandle, buffer, length)` is called
   with a **fresh, uninitialized** stack buffer, not the one that carried the
   `0x93` status query. That is a real gap in the reference: Linux's
   `HIDIOCGFEATURE` ioctl requires the caller to set `buf[0]` to the report
   id being requested before the call. `kbdspike`'s `Device.Status()` sets
   `buf[0] = 0xcc` explicitly before the read. This is the single most
   important correctness fix relative to a literal port, and it is
   untested against real hardware.
2. **`Reset()` does not also query status.** The reference's `Reset()` calls
   `GetDeviceStatus()` immediately after sending the reset command, but
   discards the result entirely (`inSet` is set from the earlier
   `PrepareAndSend` call, not from this read). `kbdspike`'s `Reset()` only
   sends the reset frame. Colour output is unaffected either way; this
   just drops one no-op device round trip.
3. **The reference's `SetAction` for a single key calls `AddV5DataBlock`
   twice at the same buffer offset**, which just overwrites the same 4
   bytes with the same values. `SetKeyColors` writes it once.
4. **No key index to key name table exists anywhere in the reference for
   vid 0d62.** `Mappings::LoadMappings` reads a user-populated
   `mappings.json` that starts empty; nothing ships default light names for
   the Darfon controller. The closest hint is `alienfx-cli`'s interactive
   naming wizard defaulting to `0x88` (136) lights when probing a `0d62`
   device (`alienfx-cli/src/main.cpp`, the `probe` subcommand). `probe`,
   `all`, `off`, and `sweep`'s default range (0-135, `kbd.DefaultKeyFirst`/
   `DefaultKeyLast`) come from that one hardcoded number, not from any
   documented zone count. `kbd.KeyNames` is an empty table; `probe` reports
   `keyTableAvailable: false` plainly rather than inventing names.
5. **Global effects (`COMMV5_setEffect`, `SetGlobalEffects`) were not
   ported.** No required verb needs it, and the reference header's own
   comment admits parts of that command are "purpose unknown" (a mask byte
   at an uncertain offset). Porting it speculatively seemed like pure added
   risk for this spike's actual goal, mapping key indices to physical keys.
6. **`SetBrightness`/`TurnOnFrame` is ported and unit-tested but not wired
   to a CLI verb**, since brightness was not in the required verb list.
   `Device.SetBrightness` exists for a future PR.
7. **Power-button/global-state persistence (`SaveLightsState`,
   `SaveLightsStateToStartup`) was not ported**, for the same reason as in
   `afxspike`: it is a separate, larger state machine the reference gates
   behind `store`/`save` flags, none of the required verbs touch it, and it
   writes to persistent device memory.

### Things I am NOT confident about, numbered

1. **Whether Feature reports are actually correct on Linux for this exact
   chip.** The reference's Windows/hidapi-libusb backend uses
   `hid_send_feature_report`/`hid_get_feature_report` for API_V5 without
   qualification, and the task's own research notes independently say
   "control via feature id 0xcc," so this is the best-supported guess
   available, but it has not been tried against `/dev/hidraw1` by me.
2. **Whether `GetFeature` with `buf[0]=0xcc` pre-set is the right fix**,
   described in divergence #1 above. It is the standard, documented Linux
   hidraw contract, but I have not verified it returns a real, non-zero
   status byte on this specific device; it could just as easily return
   `-EINVAL` if this controller's HID report descriptor does not define a
   0xcc feature report the kernel will match against, or it could hang if
   the firmware is in a bad state.
3. **Whether the reset opcode (`0x94`) actually blanks previously-set key
   colours, or only resets an internal state machine for accepting new
   commands.** Nothing in the reference clarifies this. `sweep` calls
   `SetKeyColors` per step, which calls `Reset()` first; if reset does not
   blank prior colours, `sweep` will accumulate lit keys instead of showing
   one at a time, the same open question the `elc` port flagged for its own
   `Reset()`/zone sweep.
4. **The `index + 1` wire offset in `AddV5DataBlock`.** The reference's own
   porter was unsure of it (`// NOTE: +1 because parts start from 1, 0 is
   for reset? ig`, a direct quote from the C++ source). I ported it exactly
   as coded because it is the only documented behaviour available, but "0
   is for reset" suggests wire index 0 might be a reserved/special value
   rather than a real key, meaning key index 255 (wire value 0 after
   wraparound) or some low real key index could collide with it. This
   spike does not guard against that.
5. **The 0-135 default key range (`kbd.DefaultKeyFirst`/`DefaultKeyLast`)
   is a guess sourced from one hardcoded constant in an unrelated CLI tool's
   interactive wizard**, not a documented zone count for this exact chip.
   `all`/`off` will address all 136 indices whether or not the real
   keyboard has that many; unused indices should be harmless no-ops if the
   firmware ignores out-of-range keys, but that assumption itself is
   untested.
6. **Whether `UpdateFrame` (`0x8b 0x01 0xff`) is really required after every
   `SetKeyColors` call, or only needed once per session.** The reference
   only shows it called once, after a whole batch, in commented-out example
   code; `kbdspike` sends it after every `all`/`key`/`sweep`-step write to
   stay safe, which means `sweep` sends far more update frames than the
   reference's own usage pattern implies are necessary. This is a
   conservative choice, not a verified one, and if the firmware treats
   rapid repeated updates specially (rate limiting, debounce, a wedge like
   the AW-ELC controller has), `sweep` at a short `--dwell` could be the
   first thing to find that out.
7. **No live testing was done at all.** Per the task's hard rules, this was
   built and unit-tested only, against fakes; `/dev/hidraw1` was never
   opened by me, no built binary was run, and nothing here has touched real
   hardware.
