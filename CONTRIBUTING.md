# Contributing

## Setup

```bash
git clone https://github.com/chr0nzz/omarchy-alienware.git
cd omarchy-alienware
omarchy plugin add "$PWD" --enable
cd helper && go build ./cmd/alienwarectl
```

Plugin edits hot-reload. If a change does not apply, run `omarchy restart shell`.
`quickshell log -i <instance> -t 100` shows QML errors, `quickshell list --all` prints the instance id.
Helper changes need a rebuild and `systemctl restart alienwarectl.service` before the plugin sees them.
`journalctl -u alienwarectl -f` shows daemon logs.

## Layout

| File | Role |
| --- | --- |
| `Service.qml` | One per session: polling, state file, action queue, RGB, IPC |
| `BarWidget.qml` | The bar pill, one per monitor |
| `Panel.qml` | The popup with the fan, profile, power and RGB tabs |
| `components/` | Curve editor, fan card, mode tile, readout row, slider, region row |
| `Model.js` | Pure logic: parsing, curve maths, region colours, formatting |
| `tests/` | Node tests for `Model.js` |
| `helper/cmd/alienwarectl` | Entry point for both the daemon and the CLI |
| `helper/internal/cli` | Verb dispatch and JSON output |
| `helper/internal/daemon` | D-Bus service, polkit gate, curve loop, watchdog |
| `helper/internal/hw` | sysfs reads and writes |
| `helper/internal/fan` | Pure logic: curve parsing, clamping, interpolation, hysteresis |
| `helper/internal/client` | D-Bus client used by the CLI |
| `helper/internal/elc` | AlienFX ELC protocol, the region map and the RGB session |
| `helper/internal/hidraw` | Pure-Go hidraw ioctl layer |
| `packaging/` | PKGBUILD, systemd units, D-Bus policy, polkit action and rule |

## Tests

Both suites must pass.

```bash
npm test
cd helper && go test ./...
```

The Node tests need Node 22 or newer. They load `Model.js` directly, so anything that can live in `Model.js` should, with a test next to it.

Go changes must also pass:

```bash
cd helper
go vet ./...
gofmt -l .
go test -race ./...
```

`gofmt -l .` must print nothing. The race detector matters here because the curve loop, the D-Bus method handlers and the watchdog all touch the same service state.

Pure logic belongs in `Model.js` on the QML side and `internal/fan` on the Go side, where it can be tested. The QML views and the daemon plumbing are deliberately thin.

## Hardware

The build machine is usually not the target laptop, so hardware paths cannot be tested locally. Anything that touches sysfs, the `alienware_wmi` hwmon, RAPL or `platform_profile` needs a real x15 R2 to verify.

State in the pull request what you tested on hardware and what you did not. An untested hardware path is fine to submit, an untested hardware path claimed as verified is not.

## Fan control

Any change to the fan control path must state the fail-safe reasoning in the pull request: what resets boost, on which exit paths, and what happens if the change fails halfway. The existing fail-safes are listed in [SECURITY.md](SECURITY.md). Do not remove one without a replacement.

`StopCurve` is not gated by polkit on purpose. Stopping a curve only returns control to the firmware, so a polkit failure must never be able to block it.

## Contract

The Go helper and the QML share a JSON and CLI contract. `alienwarectl status` output is parsed by `Model.js`, and `Service.qml` builds every CLI invocation. A change to one side needs the matching change to the other in the same pull request, with the Node test and the Go test both updated.

## Style

- No code comments, in any language. Name things so they explain themselves.
- No em dashes anywhere: code, docs, commit messages. Use a comma, a hyphen, or a full stop.
- Match the shell: build on the existing components and take colours, spacing, and fonts from the theme. Do not invent new styles.
- The QML calls `alienwarectl` and reads sysfs. It does not call `sudo`, `pkexec`, or write to sysfs itself.
- Every new privileged D-Bus method needs a polkit action and a gate, unless blocking it could leave the fans pinned.
- Every CLI verb prints JSON and exits non-zero on failure.
- Keep docs short. Prefer a table to a paragraph. Describe what a thing does, not why.

## Pull requests

- One change per pull request.
- Commit messages are a single line.
- Update `README.md` when behaviour, settings, keys, CLI verbs, D-Bus methods, or IPC change.
- Update `packaging/` when a unit, policy, action, or installed file changes.
- Bump `version` in `manifest.json`, `package.json`, and `pkgver` in `PKGBUILD` only when asked in review.
- Security issues go through [SECURITY.md](SECURITY.md), not a pull request.

By contributing you agree that your work is released under the [MIT License](LICENSE).
