package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
)

func run(t *testing.T, stdin string, args ...string) (int, map[string]any, string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := RunWith(Env{Stdout: &out, Stderr: &errBuf, Stdin: strings.NewReader(stdin)}, args)
	var parsed map[string]any
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
			return code, nil, out.String()
		}
	}
	return code, parsed, out.String()
}

func TestVersionPrintsJSON(t *testing.T) {
	code, parsed, raw := run(t, "", "version")
	if code != 0 {
		t.Fatalf("exit code: got %d, want 0", code)
	}
	if parsed["ok"] != true {
		t.Fatalf("ok: got %v in %s", parsed["ok"], raw)
	}
	if parsed["version"] != Version {
		t.Fatalf("version: got %v, want %q", parsed["version"], Version)
	}
	if !strings.HasSuffix(raw, "\n") {
		t.Error("output must end with a newline")
	}
}

func TestNoArgsFails(t *testing.T) {
	code, parsed, raw := run(t, "")
	if code == 0 {
		t.Fatal("want a non zero exit code")
	}
	if parsed["ok"] != false {
		t.Fatalf("ok: got %v in %s", parsed["ok"], raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("code: got %v, want %q", parsed["code"], CodeBadRequest)
	}
	if parsed["error"] == "" {
		t.Error("error must carry a message")
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := RunWith(Env{Stdout: &out, Stderr: &errBuf, Stdin: strings.NewReader("")}, []string{"help"})
	if code != 0 {
		t.Fatalf("exit code: got %d, want 0", code)
	}
	if !strings.Contains(out.String(), "alienwarectl <verb>") {
		t.Fatalf("usage missing from %q", out.String())
	}
}

type fakeCoded struct {
	msg  string
	code string
}

func (e *fakeCoded) Error() string { return e.msg }
func (e *fakeCoded) Code() string  { return e.code }

func TestEmitErrorShape(t *testing.T) {
	var out bytes.Buffer
	code := emitError(&out, &fakeCoded{msg: "no server", code: "no-openrgb"})
	if code != 1 {
		t.Fatalf("exit code: got %d, want 1", code)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if parsed["ok"] != false || parsed["error"] != "no server" || parsed["code"] != "no-openrgb" {
		t.Fatalf("got %v", parsed)
	}
	if len(parsed) != 3 {
		t.Fatalf("failure JSON must carry exactly ok, error and code, got %v", parsed)
	}
}

func TestEmitErrorWrappedCode(t *testing.T) {
	wrapped := &wrapErr{inner: &fakeCoded{msg: "inner", code: "hw-missing"}}
	var out bytes.Buffer
	emitError(&out, wrapped)
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["code"] != "hw-missing" {
		t.Fatalf("code: got %v, want hw-missing", parsed["code"])
	}
}

type wrapErr struct {
	inner error
}

func (e *wrapErr) Error() string { return "wrapped: " + e.inner.Error() }
func (e *wrapErr) Unwrap() error { return e.inner }

func TestEmitErrorPlainDefaultsToInternal(t *testing.T) {
	var out bytes.Buffer
	emitError(&out, errors.New("boom"))
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["code"] != CodeInternal {
		t.Fatalf("code: got %v, want %q", parsed["code"], CodeInternal)
	}
}

func TestEmitOKWritesOneLine(t *testing.T) {
	var out bytes.Buffer
	code := emitOK(&out, struct {
		OK bool `json:"ok"`
	}{true})
	if code != 0 {
		t.Fatalf("exit code: got %d", code)
	}
	if out.String() != "{\"ok\":true}\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestUsageListsEveryContractVerb(t *testing.T) {
	for _, verb := range []string{"status", "profile", "boost", "curve apply", "curve stop",
		"turbo", "pl", "gpu", "rgb status", "rgb set", "rgb set-all", "rgb mode",
		"rgb brightness", "rgb identify", "rgb off", "version"} {
		if !strings.Contains(Usage, verb) {
			t.Errorf("usage does not mention %q", verb)
		}
	}
}

var _ io.Writer = (*bytes.Buffer)(nil)

func TestEmitErrorMapsHardwareSentinels(t *testing.T) {
	cases := map[string]error{
		"hw-missing":    fmt.Errorf("%w: alienware_wmi hwmon not present", hw.ErrHardwareMissing),
		"not-supported": fmt.Errorf("%w: no_turbo absent", hw.ErrNotSupported),
		"bad-request":   fmt.Errorf("%w: unknown fan", hw.ErrBadRequest),
	}
	for want, err := range cases {
		var out bytes.Buffer
		emitError(&out, err)
		var parsed map[string]any
		if uerr := json.Unmarshal(out.Bytes(), &parsed); uerr != nil {
			t.Fatal(uerr)
		}
		if parsed["code"] != want {
			t.Errorf("%v: code got %v, want %q", err, parsed["code"], want)
		}
	}
}

func TestEmitErrorMapsCurveValidation(t *testing.T) {
	var out bytes.Buffer
	emitError(&out, fmt.Errorf("%w: cpu needs at least 2 points", fan.ErrInvalidCurve))
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("code: got %v, want %q", parsed["code"], CodeBadRequest)
	}
}
