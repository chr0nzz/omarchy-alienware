# Development

## Tests

```
npm test
cd helper && go test ./...
```

## Releases

Tagging `v*` builds a static `alienwarectl`, verifies it, and publishes it with checksums. Grab it
instead of building:

```
gh release download --repo chr0nzz/omarchy-alienware --pattern 'alienwarectl-*-x86_64.tar.gz'
```

`makepkg` still builds from source and is the supported path.

## Packaging

There are two PKGBUILDs and they install an identical set of eight files.

| | |
|---|---|
| `packaging/` | Builds from the git tag named by `pkgver`. Needs Go, runs the full test suite in `check()` |
| `packaging/bin/` | Downloads the release tarball and installs the prebuilt binary. Needs no toolchain |

The prebuilt package pins a version and a checksum, so it has to be pointed at each new release
after the release workflow has published it:

```
./packaging/bin/update.sh 0.3.3
```

That reads `sha256sums.txt` from the published release and rewrites `pkgver` and `sha256sums`. It
fails loudly if the release or its assets are missing, so it cannot silently pin a stale version.

## Cutting a release

1. Bump `pkgver` in `packaging/PKGBUILD`
2. Write `release-notes/vX.Y.Z.md`
3. Commit, `git tag -a vX.Y.Z`, push with `--follow-tags`
4. Wait for the release workflow, it runs the suite before publishing and will not release a red tree
5. `./packaging/bin/update.sh X.Y.Z`, then commit that
