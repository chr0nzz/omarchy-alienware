# Roadmap

Everything below was established on the author's Alienware x15 R2. Other models in the family
differ, in particular the chassis zone ids and the keyboard index map.

## Working

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

## Planned

| Item | Where it stands |
|---|---|
| `kbd brightness` | `kbd.Device.SetBrightness` exists in the device layer, no CLI verb exposes it |
| Mute indicator settings | Always on and red. No toggle, no colour picker, not persisted to state |
| RGB effects | `rgb mode` is a recognised verb that always answers `not-supported`. Only flat colours per region are wired up |
| Keyboard effects | Not started. The controller drives its own loop frame, so this is a protocol question, not a UI one |
| Per led addressing inside a ring half | The AW-ELC supports it, the CLI only exposes whole regions |

## Not possible on this hardware

| Item | Why |
|---|---|
| Fan curves that make fans quieter | The kernel exposes one additive `fanN_boost` per fan, not a curve upload. A curve can only raise fans above the firmware's own choice, never lower them. The curve editor draws that floor |
| Battery charge limit | Absent on this model |
| GPU MUX and dynamic boost | Absent on this model |
| Mute and caps lock indicator lamps | Firmware driven off HID state. They are not in the AlienFX matrix and cannot be written |
