#!/usr/bin/env bash
set -euo pipefail

openrgb_bin=${OPENRGB_BIN:-openrgb}
alienwarectl_bin=${ALIENWARECTL_BIN:-alienwarectl}
server=${ALIENWARE_RGB_SERVER:-127.0.0.1:6742}
device=${ALIENWARE_RGB_DEVICE:-0}
state_home=${XDG_STATE_HOME:-$HOME/.local/state}
out_file=${1:-$state_home/omarchy/alienware/zones.json}

red=FF0000
black=000000

device_name="device $device"
active_mode=""
zone_count=${ALIENWARE_ZONE_COUNT:-}
restored=0

die() {
    printf 'rgb-sweep: %s\n' "$1" >&2
    exit 1
}

note() {
    printf '%s\n' "$1" >&2
}

rgb() {
    "$openrgb_bin" --client "$server" --device "$device" "$@" >/dev/null 2>&1
}

paint_all() {
    rgb --color "$1"
}

paint_zone() {
    rgb --zone "$1" --color "$2"
}

restore() {
    if [ "$restored" -eq 1 ]; then
        return
    fi
    restored=1
    if [ -n "$active_mode" ] && rgb --mode "$active_mode"; then
        note "restored mode $active_mode"
    else
        paint_all "$black" || true
        note "no previous mode could be restored, the LEDs were blanked"
    fi
}

json_escape() {
    local s=$1
    s=${s//\\/\\\\}
    s=${s//\"/\\\"}
    printf '%s' "$s"
}

read_status() {
    local json
    json=$("$alienwarectl_bin" rgb status 2>/dev/null) || return 1
    case "$json" in
        *'"ok":true'*) ;;
        *) return 1 ;;
    esac
    if [ -z "$zone_count" ]; then
        zone_count=$(printf '%s' "$json" | sed -n 's/.*"zoneCount":[[:space:]]*\([0-9][0-9]*\).*/\1/p')
    fi
    if [ -z "$active_mode" ]; then
        active_mode=$(printf '%s' "$json" | sed -n 's/.*"activeMode":"\([^"]*\)".*/\1/p')
    fi
    local name
    name=$(printf '%s' "$json" | sed -n 's/.*"name":"\([^"]*\)".*"zoneCount".*/\1/p')
    if [ -n "$name" ]; then
        device_name=$name
    fi
    [ -n "$zone_count" ]
}

count_tokens() {
    awk '
    {
        n = 0
        inquote = 0
        started = 0
        for (i = 1; i <= length($0); i++) {
            ch = substr($0, i, 1)
            if (ch == "\"") {
                inquote = !inquote
                if (!started) {
                    started = 1
                    n++
                }
                continue
            }
            if (!inquote && (ch == " " || ch == "\t")) {
                started = 0
                continue
            }
            if (!started) {
                started = 1
                n++
            }
        }
        print n
    }'
}

read_listing() {
    local listing block
    listing=$("$openrgb_bin" --client "$server" --list-devices 2>/dev/null) || return 1
    block=$(printf '%s\n' "$listing" | awk -v want="$device" '
        /^[0-9]+:/ {
            split($0, parts, ":")
            inside = ((parts[1] + 0) == (want + 0))
        }
        inside { print }
    ')
    [ -n "$block" ] || return 1
    device_name=$(printf '%s\n' "$block" | head -n 1 | sed 's/^[0-9]*: *//')
    if [ -z "$active_mode" ]; then
        active_mode=$(printf '%s\n' "$block" | sed -n 's/^[[:space:]]*Modes:.*\[\([^]]*\)\].*/\1/p' | head -n 1)
        active_mode=${active_mode%\"}
        active_mode=${active_mode#\"}
    fi
    if [ -z "$zone_count" ]; then
        zone_count=$(printf '%s\n' "$block" | sed -n 's/^[[:space:]]*Zones:[[:space:]]*//p' | head -n 1 | count_tokens)
    fi
    [ -n "$zone_count" ]
}

command -v "$openrgb_bin" >/dev/null 2>&1 || die "the openrgb CLI is not on PATH"
[ -t 0 ] || die "this script is interactive and needs a terminal on stdin"

if ! read_status; then
    read_listing || die "cannot talk to the OpenRGB SDK server at $server, start openrgb-server.service first"
fi

case "$zone_count" in
    '' | *[!0-9]*) die "could not work out the zone count, set ALIENWARE_ZONE_COUNT" ;;
esac
[ "$zone_count" -gt 0 ] || die "device $device reports no zones"

trap restore EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

note "device $device: $device_name"
note "$zone_count zones to sweep"
if [ -n "$active_mode" ]; then
    note "mode $active_mode will be restored when this finishes"
else
    note "the current mode could not be read, the LEDs will be blanked at the end"
fi
note ""
note "each zone lights red in turn, type what lit up and press enter"
note "press enter with nothing typed to leave a zone unnamed"
note ""

paint_all "$black" || die "could not blank the LEDs, is the SDK server running at $server"

names=()
enabled=()
for ((i = 0; i < zone_count; i++)); do
    if ! paint_zone "$i" "$red"; then
        note "zone $i could not be lit, leaving it unnamed"
        names+=("Unknown")
        enabled+=("false")
        continue
    fi
    answer=""
    read -r -p "zone $i is lit, what is it? " answer || answer=""
    paint_zone "$i" "$black" || true
    answer=${answer#"${answer%%[![:space:]]*}"}
    answer=${answer%"${answer##*[![:space:]]}"}
    if [ -z "$answer" ]; then
        names+=("Unknown")
        enabled+=("false")
    else
        names+=("$answer")
        enabled+=("true")
    fi
done

mkdir -p "$(dirname "$out_file")"
{
    printf '{\n  "zones": [\n'
    for ((i = 0; i < zone_count; i++)); do
        sep=","
        if [ "$i" -eq $((zone_count - 1)) ]; then
            sep=""
        fi
        printf '    { "index": %d, "name": "%s", "enabled": %s }%s\n' \
            "$i" "$(json_escape "${names[i]}")" "${enabled[i]}" "$sep"
    done
    printf '  ]\n}\n'
} >"$out_file"

note ""
note "wrote $out_file"
note "merge its zones array into the omarchy alienware state.json to use it"
