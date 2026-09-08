# Security

## Reporting a vulnerability

Report privately through [GitHub private vulnerability reporting](https://github.com/chr0nzz/omarchy-alienware/security/advisories/new).
Do not open a public issue for anything that could be exploited.

Include what you found, how to reproduce it, and the plugin version, the `alienwarectl version` output, and your kernel version.
You will get an acknowledgement within 7 days and a fix or a decision within 30 days.

Report anything that leaves the fans off or pinned, or that lets a caller past the polkit gate, even if you are not sure it is exploitable.

## Supported versions

Only the latest release on `main` receives fixes.

## Privilege boundary

| Part | Runs as | Interface |
| --- | --- | --- |
| Bar widget and panel | Your user | Reads sysfs, runs `alienwarectl` |
| `alienwarectl daemon` | root | Owns `org.xyzlab.Alienware1` on the D-Bus system bus, writes sysfs |
| `alienwarectl` CLI | Your user | D-Bus client of the daemon |
| RGB | Your user | Direct HID feature reports to `/dev/hidraw0` |

There are no setuid binaries, no `sudo` calls, and no root shells. The plugin never writes sysfs itself.

The daemon unit runs with `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`, `RestrictAddressFamilies=AF_UNIX AF_NETLINK` and a capability bounding set of `CAP_DAC_OVERRIDE` alone. It takes no network input. Its only inputs are D-Bus method arguments and the values it reads from sysfs.

## polkit gate

| D-Bus method | Action |
| --- | --- |
| `Status` | None, read only |
| `SetProfile` | `org.xyzlab.alienware.set-profile` |
| `SetBoost` | `org.xyzlab.alienware.set-fan` |
| `ApplyCurve` | `org.xyzlab.alienware.set-fan` and `org.xyzlab.alienware.set-profile` |
| `StopCurve` | None, deliberately |
| `SetTurbo` | `org.xyzlab.alienware.set-power` |
| `SetPowerLimit` | `org.xyzlab.alienware.set-power` |

The shipped rule allows an active local session and requires an admin password for anything else. The D-Bus policy lets any user send to the interface, because authorization is polkit's job and not the bus policy's.

`StopCurve` is not gated. Stopping a curve resets fan boost to 0 and hands the fans back to the firmware, so a polkit failure must never be able to block it. Gating it would turn an authorization outage into a thermal problem.

## Thermal risk

The honest worst case here is thermal, not data. A bug that pins the fans off, or a control loop that dies leaving boost wherever it stopped, is the failure that matters.

Fan boost is additive. A curve raises the fans above the firmware's own choice and cannot lower them, so a stuck boost value leaves the fans louder than needed rather than off. The firmware keeps its own thermal control at all times.

Fail-safes, all of which write boost 0:

| Trigger | Fail-safe |
| --- | --- |
| Daemon start | Boost reset before the bus name is claimed |
| Curve stop, or a replacement curve | Boost reset, previous platform profile restored |
| Curve loop exit or panic | Boost reset |
| Temperature unreadable for a fan | That fan's boost reset |
| SIGTERM, SIGINT | Boost reset, then exit |
| System bus connection drops | Daemon stops rather than keep driving the fans |
| Any exit, including SIGKILL | `ExecStopPost=/usr/bin/alienwarectl reset-fans` |
| Control loop hangs | `WatchdogSec=10` restarts the service |

Curve JSON is clamped and validated before it reaches the fans: temperature 0 to 110, boost 0 to 255, interval 1 to 30 seconds, hysteresis 0 to 15, 2 to 8 points per fan, sorted and deduplicated. `SetBoost` is refused while a curve is running.

## RGB

RGB deliberately runs unprivileged. The shipped `71-alienware-aw-elc.rules` tags the AW-ELC HID node `uaccess`, so your logged-in session gets an ACL on it and lighting is driven as you. It never touches the daemon.

Lighting now speaks the AlienFX protocol straight to `/dev/hidraw0`. There is no server, no socket and no network surface at all. The controller firmware wedges if two processes open that HID node at once, so only one writer should run at a time.

## What is written to disk

| Data | Where | Notes |
| --- | --- | --- |
| Fan curve, RGB zone names, colour, mode, brightness | `$XDG_STATE_HOME/omarchy/alienware/state.json` | Defaults to `~/.local/state` |
| Widget settings | `~/.config/omarchy/shell.json` | Written by the settings editor and `omarchy bar set` |

Neither file holds a credential. There are no credentials anywhere in this plugin: no server accounts, no tokens, no keys.

## Trust model

- Anything running as your user can call the `omarchy-shell alienware` IPC and the `alienwarectl` CLI, so it can change profiles, fans, turbo, power limits and lighting within the polkit rules.
- The polkit rule grants an active local session silently. That is the same level of trust the shell already has.
- Fan curve JSON given to `curve apply` is read from a file or stdin as your user. It is data, not code, and every field is clamped.
- Power limit writes are refused by the firmware when it does not permit them. The daemon reports the refusal and does not retry.

## Out of scope

- The `alienware_wmi` kernel driver and anything it exposes in sysfs. Report those to the [Linux kernel platform drivers list](https://lore.kernel.org/platform-driver-x86/).
- `nvidia-smi`, the Omarchy shell, polkit, and systemd.
