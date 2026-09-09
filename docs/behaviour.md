# Behaviour

How the plugin behaves around suspend, audio, thermal profiles and failure.

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

## Mute indicator

The two mute keys follow PipeWire. `pactl subscribe` provides the events, `wpctl get-volume` reads
the state, and a muted sink paints `volmute` (index 16) while a muted source paints `micmute`
(index 19) red.

It is an overlay, applied at write time on top of the effective key map and never stored into the
saved colours, so muting cannot eat whatever colour you picked for those two keys. The indicator
shows even when that key is individually switched off, because an indicator you have turned off is
useless, but it stays dark when the master lights toggle is off. Reads are debounced by 400ms, since
this controller wedges when transactions arrive too close together.

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

## Fail-safe

The daemon writes boost 0 on start, when a curve stops, on SIGTERM and SIGINT, and from
`ExecStopPost` so a crash or SIGKILL cannot leave the fans pinned. A systemd watchdog restarts a
hung control loop.
