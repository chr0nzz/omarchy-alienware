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
