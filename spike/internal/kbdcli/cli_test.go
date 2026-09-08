package kbdcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func run(args ...string) (int, string) {
	var out, errOut bytes.Buffer
	code := RunWith(Env{Stdout: &out, Stderr: &errOut}, args)
	return code, out.String()
}

func TestNoVerbFails(t *testing.T) {
	code, out := run()
	if code == 0 {
		t.Fatalf("expected non-zero exit with no verb")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestUnknownVerbFails(t *testing.T) {
	code, out := run("frobnicate")
	if code == 0 {
		t.Fatalf("expected non-zero exit for an unknown verb")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestHelpSucceeds(t *testing.T) {
	code, out := run("help")
	if code != 0 {
		t.Fatalf("help exited %d, want 0", code)
	}
	if !strings.Contains(out, "kbdspike") {
		t.Fatalf("help output did not mention kbdspike: %q", out)
	}
}

func TestAllRequiresColorArg(t *testing.T) {
	code, out := run("all")
	if code == 0 {
		t.Fatalf("expected non-zero exit with a missing colour argument")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestAllRejectsBadColor(t *testing.T) {
	code, out := run("all", "notacolor")
	if code == 0 {
		t.Fatalf("expected non-zero exit for a malformed colour")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestKeyRequiresTwoArgs(t *testing.T) {
	code, out := run("key", "4")
	if code == 0 {
		t.Fatalf("expected non-zero exit with a missing colour argument")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestKeyRejectsBadIndex(t *testing.T) {
	code, out := run("key", "-1", "ff8800")
	if code == 0 {
		t.Fatalf("expected non-zero exit for a negative key index")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestSweepRequiresTwoArgs(t *testing.T) {
	code, out := run("sweep", "0")
	if code == 0 {
		t.Fatalf("expected non-zero exit with a missing last key index")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestSweepRejectsBadDwell(t *testing.T) {
	code, out := run("sweep", "0", "1", "--dwell=notaduration")
	if code == 0 {
		t.Fatalf("expected non-zero exit for a malformed dwell")
	}
	assertFailureJSON(t, out, CodeBadRequest)
}

func TestMissingDeviceFailsCleanly(t *testing.T) {
	code, out := run("probe", "--device=/dev/does-not-exist-kbdspike-test")
	if code == 0 {
		t.Fatalf("expected non-zero exit when the device path does not exist")
	}
	assertFailureJSON(t, out, CodeInternal)
}

func TestOffFailsCleanlyWithoutDevice(t *testing.T) {
	code, out := run("off", "--device=/dev/does-not-exist-kbdspike-test")
	if code == 0 {
		t.Fatalf("expected non-zero exit when the device path does not exist")
	}
	assertFailureJSON(t, out, CodeInternal)
}

func TestSplitFlags(t *testing.T) {
	positional, flags := splitFlags([]string{"sweep", "0", "19", "--dwell=500ms", "--device=/dev/hidraw1", "--verbose"})
	if strings.Join(positional, ",") != "sweep,0,19" {
		t.Fatalf("positional = %v, want [sweep 0 19]", positional)
	}
	if flags["dwell"] != "500ms" || flags["device"] != "/dev/hidraw1" {
		t.Fatalf("flags = %v", flags)
	}
	if _, ok := flags["verbose"]; !ok {
		t.Fatalf("expected a bare --verbose flag to be recorded")
	}
}

func TestParseHexColor(t *testing.T) {
	r, g, b, err := parseHexColor("#Ff8800")
	if err != nil {
		t.Fatalf("parseHexColor error = %v", err)
	}
	if r != 0xff || g != 0x88 || b != 0x00 {
		t.Fatalf("parsed = %02x%02x%02x, want ff8800", r, g, b)
	}
	if hexColor(r, g, b) != "ff8800" {
		t.Fatalf("hexColor round trip = %s, want ff8800", hexColor(r, g, b))
	}
}

func TestParseHexColorRejectsWrongLength(t *testing.T) {
	if _, _, _, err := parseHexColor("ff88"); err == nil {
		t.Fatalf("expected an error for a short colour string")
	}
}

func assertFailureJSON(t *testing.T, out string, wantCode string) {
	t.Helper()
	var f struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("could not parse JSON output %q: %v", out, err)
	}
	if f.OK {
		t.Fatalf("expected ok=false in output %q", out)
	}
	if f.Error == "" {
		t.Fatalf("expected a non-empty error message in output %q", out)
	}
	if f.Code != wantCode {
		t.Fatalf("code = %q, want %q in output %q", f.Code, wantCode, out)
	}
}
