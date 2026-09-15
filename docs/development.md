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

Both packages pin an immutable reference and a SHA-256, so neither can build code that was swapped
out later. The source package names a 40-character commit and the archive digest for it, never a
tag, because a tag can be moved. Both have to be pointed at each new release after the release
workflow has published it:

```
./packaging/bin/update.sh 0.3.3
```

That resolves the commit behind the tag, downloads the archive for that exact commit, hashes it, and
rewrites `pkgver`, `_commit` and `sha256sums` in both `PKGBUILD` files. Every step exits non-zero on
failure and the writes are read back, so it cannot silently pin a stale or unverified source.

## Cutting a release

Four files carry the version and CI fails if they disagree: `manifest.json`, `package.json`,
`helper/internal/cli/cli.go` and `packaging/PKGBUILD`. `packaging/bin/PKGBUILD` is the exception,
it trails by one commit because its checksum only exists once the release is published.

1. Bump the version in all four files
2. Write `release-notes/vX.Y.Z.md`
3. Commit, `git tag -a vX.Y.Z`, push with `--follow-tags`
4. Wait for the release workflow, it runs the suite before publishing and will not release a red tree
5. `./packaging/bin/update.sh X.Y.Z`, then commit that

## Marketplace

Listing and updating are two different forms.

| Step | Form | When |
|---|---|---|
| First listing | `submit-plugin` | Once, by hand. Needs repository URL, category and up to three tags |
| Every release after that | `verify-plugin` | Opened by the release workflow |

The verify form only accepts a plugin that already has a listing, so the workflow skips it until the
repository variable `MARKETPLACE_LISTED` is set to `true`. Set that after the first listing is
approved, and add a `MARKETPLACE_TOKEN` secret that can open issues on the marketplace repository.
