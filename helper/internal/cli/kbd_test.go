package cli

import "testing"

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
