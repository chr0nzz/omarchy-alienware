## What

One or two sentences on what changes.

## Why

Link the issue, or say what was broken or missing.

## Tested on hardware

What you ran on a real x15 R2, and what you could not test. Say "nothing, build machine only" if that is the case.

## Fail-safe

Only for changes to the fan control path. What resets boost, on which exit paths, and what happens if this change fails halfway.

## Checklist

- [ ] One change per pull request
- [ ] `npm test` passes, with a test added or updated for every `Model.js` change
- [ ] `go test ./...` and `go test -race ./...` pass in `helper/`
- [ ] `go vet ./...` passes and `gofmt -l .` is empty
- [ ] The JSON and CLI contract between the helper and the QML stays in sync, both sides changed together
- [ ] New privileged D-Bus methods have a polkit action and a gate
- [ ] Checked by hand in a running Omarchy shell
- [ ] `README.md` updated if behaviour, settings, keys, CLI verbs, D-Bus methods, or IPC changed
- [ ] `packaging/` updated if a unit, policy, action, or installed file changed
- [ ] No code comments, no em dashes
- [ ] Commit messages are a single line
