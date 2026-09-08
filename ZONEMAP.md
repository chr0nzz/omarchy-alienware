# Alienware x15 R2 lighting map

Discovered empirically on 2026-09-08 by bisecting hardware zone ids over the
AlienFX ELC protocol on /dev/hidraw0, confirmed visually at every step.

## AW-ELC chassis controller

USB 187c:0550, /dev/hidraw0, readable and writable by the logged in user
through a uaccess ACL, so no root is needed.

| Hardware zone ids | Region |
| --- | --- |
| 0 | power button |
| 1 | lid logo |
| 2-7 | inert |
| 8-15 | ring, first half |
| 16-23 | ring, second half |
| 24-31 | inert |

Four independently addressable regions. The ring halves each cover eight
ids, so per LED addressing inside a half is available if wanted.

The keyboard is not driven by this controller. It stayed on its own stored
colour through every id from 0 to 31.

## Darfon keyboard controller

USB 0d62:babc, /dev/hidraw1, root only, no ACL. Holds a persistent colour
that survives AW-ELC writes, OpenRGB writes, USB resets and server restarts.
Reaching it needs AlienFX APIv5 and a udev rule for user access.

## Why OpenRGB could not drive this machine

OpenRGB reads platform id 0x306, which is absent from its tables, so it falls
back to sequential hardware zone ids 0 to 19. The real layout is sparse and
spans 0 to 23, so most OpenRGB writes addressed ids that do not exist on this
hardware. Only the power button, at id 0, ever overlapped.
