# alienwarectl

```
alienwarectl status
alienwarectl profile balanced
alienwarectl boost cpu 120
alienwarectl curve apply curve.json
alienwarectl curve stop
alienwarectl turbo on
alienwarectl pl 1 65
alienwarectl rgb status
alienwarectl rgb set logo ff0044
alienwarectl rgb set-all ff8800
alienwarectl rgb set-map logo=ff0000,power=00ff00
alienwarectl rgb brightness 75
alienwarectl rgb identify ring-top
alienwarectl rgb off
alienwarectl rgb reset
alienwarectl kbd status
alienwarectl kbd set-map 0=ff0000,4=00ff00
alienwarectl kbd set-all ff8800
alienwarectl kbd off
```

`rgb set`, `rgb identify` and the `region=colour` pairs in `rgb set-map` take a region id from the
[RGB regions](#rgb-regions) table: `power`, `logo`, `ring-top` or `ring-bottom`. An unknown region
id is a `bad-request` error that names the valid ones. `rgb mode` is recognised but always answers
`not-supported`, lighting effects are not implemented yet.

`kbd` verbs are daemon clients like `profile`, `boost` and `pl`: they never open the keyboard's
hidraw node directly, only the root daemon does. `idx` in `kbd set-map` is the raw wire key index,
0-135.

Every command prints JSON and exits non-zero on failure.
