package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/kbd"
)

func TestParseKeyColorMapDisplayParsesPairs(t *testing.T) {
	got, err := parseKeyColorMapDisplay("0=ff0000,4=00ff00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["0"] != "ff0000" || got["4"] != "00ff00" {
		t.Fatalf("got %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
}

func TestParseKeyColorMapDisplayRejectsMalformedEntry(t *testing.T) {
	if _, err := parseKeyColorMapDisplay("0-ff0000"); err == nil {
		t.Fatal("want an error for a missing =")
	}
}

func TestParseKeyColorMapDisplayRejectsNonNumericIndex(t *testing.T) {
	if _, err := parseKeyColorMapDisplay("power=ff0000"); err == nil {
		t.Fatal("want an error for a non numeric key index")
	}
}

func TestParseKeyColorMapDisplayRejectsBadColor(t *testing.T) {
	if _, err := parseKeyColorMapDisplay("0=notacolor"); err == nil {
		t.Fatal("want an error for a malformed colour")
	}
}

func TestParseKeyColorMapDisplayRejectsEmpty(t *testing.T) {
	if _, err := parseKeyColorMapDisplay(""); err == nil {
		t.Fatal("want an error for an empty map")
	}
}

func TestKbdUnknownVerb(t *testing.T) {
	code, parsed, raw := run(t, "", "kbd", "bogus")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestKbdSetMapRejectsMalformedEntryBeforeContactingTheDaemon(t *testing.T) {
	code, parsed, raw := run(t, "", "kbd", "set-map", "power=ff0000")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestIdentifyKeyMapLightsExactlyOneKeyWhite(t *testing.T) {
	idx, payload, err := identifyKeyMap("12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 12 {
		t.Fatalf("index: got %d, want 12", idx)
	}
	pairs := strings.Split(payload, ",")
	want := kbd.DefaultKeyLast - kbd.DefaultKeyFirst + 1
	if len(pairs) != want {
		t.Fatalf("got %d pairs, want %d, identify must blank every key", len(pairs), want)
	}
	lit := []string{}
	for _, pair := range pairs {
		if !strings.HasSuffix(pair, "=000000") {
			lit = append(lit, pair)
		}
	}
	if len(lit) != 1 || lit[0] != "12=ffffff" {
		t.Fatalf("lit keys: got %v, want exactly [12=ffffff]", lit)
	}
	if pairs[0] != "0=000000" || pairs[want-1] != strconv.Itoa(kbd.DefaultKeyLast)+"=000000" {
		t.Fatalf("the map must span %d-%d, got first %q last %q", kbd.DefaultKeyFirst, kbd.DefaultKeyLast, pairs[0], pairs[want-1])
	}
}

func TestIdentifyKeyMapCoversTheWholeRange(t *testing.T) {
	for _, idx := range []int{kbd.DefaultKeyFirst, kbd.DefaultKeyLast} {
		got, payload, err := identifyKeyMap(strconv.Itoa(idx))
		if err != nil {
			t.Fatalf("index %d: unexpected error: %v", idx, err)
		}
		if got != idx {
			t.Fatalf("index %d: got %d", idx, got)
		}
		if !strings.Contains(payload, strconv.Itoa(idx)+"=ffffff") {
			t.Fatalf("index %d is not lit in %q", idx, payload)
		}
	}
}

func TestIdentifyKeyMapRejectsNonNumericIndex(t *testing.T) {
	if _, _, err := identifyKeyMap("esc"); err == nil {
		t.Fatal("want an error for a non numeric key index")
	}
}

func TestIdentifyKeyMapRejectsOutOfRangeIndex(t *testing.T) {
	for _, arg := range []string{"-1", strconv.Itoa(kbd.DefaultKeyLast + 1), "200"} {
		if _, _, err := identifyKeyMap(arg); err == nil {
			t.Fatalf("want an error for %q", arg)
		}
	}
}

func TestKbdIdentifyRejectsNonNumericIndexBeforeContactingTheDaemon(t *testing.T) {
	code, parsed, raw := run(t, "", "kbd", "identify", "esc")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestKbdIdentifyRejectsOutOfRangeIndexBeforeContactingTheDaemon(t *testing.T) {
	code, parsed, raw := run(t, "", "kbd", "identify", strconv.Itoa(kbd.DefaultKeyLast+1))
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestKbdIdentifyNeedsExactlyOneArgument(t *testing.T) {
	code, parsed, raw := run(t, "", "kbd", "identify")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestUsageListsKbdIdentify(t *testing.T) {
	if !strings.Contains(Usage, "kbd identify <idx>") {
		t.Fatal("the usage text must document kbd identify")
	}
}
