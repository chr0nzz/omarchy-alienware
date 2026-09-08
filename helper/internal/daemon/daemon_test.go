package daemon

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readValue(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

func fakeReader(t *testing.T) *hw.Reader {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "sys/class/hwmon/hwmon4/name", "alienware_wmi\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan1_input", "2100\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan1_boost", "0\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan2_input", "1900\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan2_boost", "0\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/temp1_input", "80000\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/temp2_input", "40000\n")
	writeFile(t, root, "sys/class/dmi/id/product_name", "Alienware x15 R2\n")
	writeFile(t, root, "sys/firmware/acpi/platform_profile", "balanced\n")
	writeFile(t, root, "sys/firmware/acpi/platform_profile_choices",
		"low-power quiet balanced balanced-performance performance custom\n")
	writeFile(t, root, "sys/devices/system/cpu/intel_pstate/no_turbo", "0\n")
	writeFile(t, root, "sys/class/powercap/intel-rapl:0/constraint_0_power_limit_uw", "65000000\n")
	return &hw.Reader{
		Root:      root,
		RunNvidia: func() (string, error) { return "22.40, [N/A], 90.00, 140.00\n", nil },
	}
}

func quietLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestStatusJSONShape(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	payload, derr := svc.Status()
	if derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		t.Fatalf("status is not valid JSON: %v", err)
	}
	for _, key := range []string{"ok", "ts", "model", "hwmon", "fans", "temps",
		"profile", "turbo", "power", "gpu", "curve", "warnings"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("status is missing the %q key", key)
		}
	}

	var s Status
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatalf("status does not decode into Status: %v", err)
	}
	if !s.OK {
		t.Error("ok must be true")
	}
	if s.TS <= 0 {
		t.Error("ts must be a unix timestamp")
	}
	if len(s.Fans) != 2 {
		t.Fatalf("fans: got %d, want exactly 2", len(s.Fans))
	}
	if s.Fans[0].ID != "cpu" || s.Fans[1].ID != "gpu" {
		t.Errorf("fan ids: got %q and %q", s.Fans[0].ID, s.Fans[1].ID)
	}
	if s.Curve.Active {
		t.Error("curve should start inactive")
	}

	var fanRaw []map[string]json.RawMessage
	if err := json.Unmarshal(raw["fans"], &fanRaw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "index", "label", "rpm", "max", "boost", "percent"} {
		if _, ok := fanRaw[0][key]; !ok {
			t.Errorf("fan object is missing the %q key", key)
		}
	}

	var profileRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["profile"], &profileRaw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"current", "choices", "ppdRunning", "gmodeForced", "writable"} {
		if _, ok := profileRaw[key]; !ok {
			t.Errorf("profile object is missing the %q key", key)
		}
	}

	var gpuRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["gpu"], &gpuRaw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"available", "draw", "limit", "defaultLimit", "maxLimit", "limitWritable"} {
		if _, ok := gpuRaw[key]; !ok {
			t.Errorf("gpu object is missing the %q key", key)
		}
	}
	if string(gpuRaw["limit"]) != "null" {
		t.Errorf("an unreadable gpu limit must serialise as null, got %s", gpuRaw["limit"])
	}

	var curveRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["curve"], &curveRaw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"active", "interval", "hysteresis", "cpu", "gpu"} {
		if _, ok := curveRaw[key]; !ok {
			t.Errorf("curve object is missing the %q key", key)
		}
	}
	if string(curveRaw["cpu"]) != "[]" {
		t.Errorf("an unset curve must serialise as an empty array, got %s", curveRaw["cpu"])
	}
	var warnRaw []string
	if err := json.Unmarshal(raw["warnings"], &warnRaw); err != nil {
		t.Errorf("warnings must serialise as an array of strings, got %s", raw["warnings"])
	}
}

func TestStatusOmitsUnreadableTemps(t *testing.T) {
	reader := fakeReader(t)
	if err := os.Remove(filepath.Join(reader.Root, "sys/class/hwmon/hwmon4/temp2_input")); err != nil {
		t.Fatal(err)
	}
	svc := NewService(reader, nil, quietLogger())
	payload, derr := svc.Status()
	if derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	var s Status
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Temps["gpu"]; ok {
		t.Error("an unreadable temp must be omitted, not reported as zero")
	}
	if s.Temps["cpu"] != 80 {
		t.Errorf("cpu temp: got %d, want 80", s.Temps["cpu"])
	}
	if len(s.Warnings) == 0 {
		t.Error("a missing temp must add a warning")
	}
}

func TestCurveDrivesBoostAndFailsSafeOnStop(t *testing.T) {
	reader := fakeReader(t)
	svc := NewService(reader, nil, quietLogger())
	curve := fan.Curve{
		Interval:   1,
		Hysteresis: 0,
		CPU:        []fan.Point{{Temp: 40, Boost: 0}, {Temp: 80, Boost: 200}},
		GPU:        []fan.Point{{Temp: 40, Boost: 0}, {Temp: 80, Boost: 200}},
	}
	if err := svc.StartCurve(curve); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan1_boost") == "200" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan1_boost"); got != "200" {
		t.Fatalf("cpu boost at 80C: got %q, want 200", got)
	}
	if got := readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan2_boost"); got != "0" {
		t.Fatalf("gpu boost at 40C: got %q, want 0", got)
	}
	if got := readValue(t, reader.Root, "sys/firmware/acpi/platform_profile"); got != "custom" {
		t.Fatalf("an active curve must select the custom profile, got %q", got)
	}
	if !svc.curveStatus().Active {
		t.Error("curve should report active")
	}

	svc.Stop()

	if got := readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan1_boost"); got != "0" {
		t.Fatalf("stopping the curve must reset the cpu boost, got %q", got)
	}
	if got := readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan2_boost"); got != "0" {
		t.Fatalf("stopping the curve must reset the gpu boost, got %q", got)
	}
	if got := readValue(t, reader.Root, "sys/firmware/acpi/platform_profile"); got != "balanced" {
		t.Fatalf("stopping the curve must restore the previous profile, got %q", got)
	}
	if svc.curveStatus().Active {
		t.Error("curve should report inactive after stop")
	}
}

func TestCurveFailsSafeWhenTemperatureIsUnreadable(t *testing.T) {
	reader := fakeReader(t)
	writeFile(t, reader.Root, "sys/class/hwmon/hwmon4/fan1_boost", "255\n")
	if err := os.Remove(filepath.Join(reader.Root, "sys/class/hwmon/hwmon4/temp1_input")); err != nil {
		t.Fatal(err)
	}
	svc := NewService(reader, nil, quietLogger())
	svc.tick(
		map[string]*fan.State{"cpu": fan.NewState(0), "gpu": fan.NewState(0)},
		map[string][]fan.Point{
			"cpu": {{Temp: 0, Boost: 255}, {Temp: 110, Boost: 255}},
			"gpu": {{Temp: 0, Boost: 0}, {Temp: 110, Boost: 0}},
		},
	)
	if got := readValue(t, reader.Root, "sys/class/hwmon/hwmon4/fan1_boost"); got != "0" {
		t.Fatalf("an unreadable temperature must fall back to boost 0, got %q", got)
	}
}

func TestSetBoostRejectedWhileCurveIsActive(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	curve := fan.Curve{Interval: 30, Hysteresis: 0,
		CPU: []fan.Point{{Temp: 40, Boost: 0}, {Temp: 80, Boost: 10}},
		GPU: []fan.Point{{Temp: 40, Boost: 0}, {Temp: 80, Boost: 10}}}
	if err := svc.StartCurve(curve); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()
	derr := svc.SetBoost("cpu", 100, dbus.Sender(":1.1"))
	if derr == nil {
		t.Fatal("want a rejection while a curve is running")
	}
	if derr.Name != ErrNameBadRequest {
		t.Fatalf("error name: got %q, want %q", derr.Name, ErrNameBadRequest)
	}
}

func TestApplyCurveRejectsBadJSON(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	derr := svc.ApplyCurve(`{"cpu":[{"temp":40,"boost":0}]}`, dbus.Sender(":1.1"))
	if derr == nil {
		t.Fatal("want a rejection")
	}
	if derr.Name != ErrNameBadRequest {
		t.Fatalf("error name: got %q, want %q", derr.Name, ErrNameBadRequest)
	}
}

type denyAll struct{}

func (denyAll) Authorize(dbus.Sender, string) error {
	return errDenied("not authorized for the test")
}

type onlyFan struct{}

func (onlyFan) Authorize(_ dbus.Sender, action string) error {
	if action == ActionSetFan {
		return nil
	}
	return errDenied("not authorized for " + action)
}

func TestDeniedErrorNameEndsInDenied(t *testing.T) {
	svc := NewService(fakeReader(t), denyAll{}, quietLogger())
	for name, derr := range map[string]*dbus.Error{
		"SetProfile":    svc.SetProfile("performance", ":1.1"),
		"SetBoost":      svc.SetBoost("cpu", 10, ":1.1"),
		"ApplyCurve":    svc.ApplyCurve("{}", ":1.1"),
		"SetTurbo":      svc.SetTurbo(true, ":1.1"),
		"SetPowerLimit": svc.SetPowerLimit(0, 45, ":1.1"),
	} {
		if derr == nil {
			t.Errorf("%s: want a denial", name)
			continue
		}
		if !strings.HasSuffix(derr.Name, ".Denied") {
			t.Errorf("%s: error name %q must end in .Denied", name, derr.Name)
		}
	}
}

func TestErrorMapping(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	if derr := svc.SetProfile("nonsense", ":1.1"); derr == nil || derr.Name != ErrNameBadRequest {
		t.Errorf("unknown profile: got %v", derr)
	}
	if derr := svc.SetBoost("middle", 10, ":1.1"); derr == nil || derr.Name != ErrNameBadRequest {
		t.Errorf("unknown fan: got %v", derr)
	}
	if derr := svc.SetPowerLimit(1, 45, ":1.1"); derr == nil || derr.Name != ErrNameHwMissing {
		t.Errorf("absent constraint: got %v", derr)
	}
}

func TestWatchdogIntervalIsHalf(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "10000000")
	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()))
	if got := watchdogInterval(); got != 5*time.Second {
		t.Fatalf("got %v, want 5s", got)
	}
}

func TestWatchdogIntervalDisabled(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "")
	if got := watchdogInterval(); got != 0 {
		t.Errorf("unset: got %v, want 0", got)
	}
	t.Setenv("WATCHDOG_USEC", "not a number")
	if got := watchdogInterval(); got != 0 {
		t.Errorf("garbage: got %v, want 0", got)
	}
	t.Setenv("WATCHDOG_USEC", "10000000")
	t.Setenv("WATCHDOG_PID", "1")
	if got := watchdogInterval(); got != 0 {
		t.Errorf("another pid: got %v, want 0", got)
	}
}

func TestNotifyWithoutSocketIsANoop(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	if err := notify("READY=1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotifyWritesToUnixgramSocket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notify.sock")
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Skipf("unixgram sockets are unavailable here: %v", err)
	}
	defer conn.Close()

	t.Setenv("NOTIFY_SOCKET", path)
	if err := notify("READY=1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 64)
	n, _, err := conn.ReadFromUnix(b)
	if err != nil {
		t.Fatalf("did not receive the notification: %v", err)
	}
	if string(b[:n]) != "READY=1" {
		t.Fatalf("got %q, want READY=1", string(b[:n]))
	}
}

func TestNotifyAbstractSocketNameIsTranslated(t *testing.T) {
	name := "@alienwarectl-test-notify"
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: "\x00" + name[1:], Net: "unixgram"})
	if err != nil {
		t.Skipf("abstract unixgram sockets are unavailable here: %v", err)
	}
	defer conn.Close()

	t.Setenv("NOTIFY_SOCKET", name)
	if err := notify("WATCHDOG=1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 64)
	n, _, err := conn.ReadFromUnix(b)
	if err != nil {
		t.Fatalf("did not receive the notification: %v", err)
	}
	if string(b[:n]) != "WATCHDOG=1" {
		t.Fatalf("got %q, want WATCHDOG=1", string(b[:n]))
	}
}

func TestStopCurveNeedsNoAuthorization(t *testing.T) {
	svc := NewService(fakeReader(t), denyAll{}, quietLogger())
	if derr := svc.StopCurve(":1.1"); derr != nil {
		t.Fatalf("stopping a curve returns control to firmware and must never be denied, got %v", derr)
	}
}

func TestApplyCurveNeedsProfilePermissionToo(t *testing.T) {
	svc := NewService(fakeReader(t), onlyFan{}, quietLogger())
	derr := svc.ApplyCurve(`{"interval":2,"hysteresis":3,"cpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}],"gpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}]}`, ":1.1")
	if derr == nil {
		t.Fatal("set-fan alone must not be enough to change the platform profile through a curve")
	}
	if !strings.HasSuffix(derr.Name, ".Denied") {
		t.Fatalf("error name %q must end in .Denied", derr.Name)
	}
}
