package hw

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func fakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, root, "sys/class/hwmon/hwmon0/name", "coretemp\n")
	writeFile(t, root, "sys/class/hwmon/hwmon0/temp1_input", "50000\n")

	writeFile(t, root, "sys/class/hwmon/hwmon4/name", "alienware_wmi\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan1_input", "2100\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan1_boost", "0\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan2_input", "1900\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/fan2_boost", "0\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/temp1_input", "47000\n")
	writeFile(t, root, "sys/class/hwmon/hwmon4/temp2_input", "43000\n")

	writeFile(t, root, "sys/class/hwmon/hwmon5/name", "dell_smm\n")
	writeFile(t, root, "sys/class/hwmon/hwmon5/temp3_input", "44000\n")
	writeFile(t, root, "sys/class/hwmon/hwmon5/temp6_input", "45000\n")

	writeFile(t, root, "sys/class/dmi/id/product_name", "Alienware x15 R2\n")
	writeFile(t, root, "sys/firmware/acpi/platform_profile", "balanced\n")
	writeFile(t, root, "sys/firmware/acpi/platform_profile_choices",
		"low-power quiet balanced balanced-performance performance custom\n")
	writeFile(t, root, "sys/module/alienware_wmi/parameters/force_gmode", "N\n")
	writeFile(t, root, "sys/devices/system/cpu/intel_pstate/no_turbo", "0\n")
	writeFile(t, root, "sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq", "2100000\n")
	writeFile(t, root, "sys/devices/system/cpu/cpu1/cpufreq/scaling_cur_freq", "4700000\n")

	writeFile(t, root, "sys/class/powercap/intel-rapl:0/constraint_0_power_limit_uw", "65000000\n")
	writeFile(t, root, "sys/class/powercap/intel-rapl:0/constraint_0_name", "long_term\n")
	writeFile(t, root, "sys/class/powercap/intel-rapl:0/constraint_1_power_limit_uw", "140000000\n")
	writeFile(t, root, "sys/class/powercap/intel-rapl:0/constraint_2_power_limit_uw", "215000000\n")

	return root
}

func fakeReader(t *testing.T) *Reader {
	t.Helper()
	return &Reader{
		Root: fakeRoot(t),
		RunNvidia: func() (string, error) {
			return "22.40, 55.00, 90.00, 140.00\n", nil
		},
	}
}

func TestFindHwmonByName(t *testing.T) {
	r := fakeReader(t)
	dir, err := r.FindHwmon(AlienwareHwmonName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(dir) != "hwmon4" {
		t.Fatalf("resolved %s, want hwmon4", dir)
	}
	if _, err := r.FindHwmon("nothing_here"); err == nil {
		t.Fatal("want an error for a missing hwmon name")
	}
}

func TestSnapshotFullTree(t *testing.T) {
	r := fakeReader(t)
	s := r.Snapshot()

	if s.Model != "Alienware x15 R2" {
		t.Errorf("model: got %q", s.Model)
	}
	if filepath.Base(s.Hwmon) != "hwmon4" {
		t.Errorf("hwmon: got %q", s.Hwmon)
	}
	if len(s.Fans) != 2 {
		t.Fatalf("fans: got %d, want 2", len(s.Fans))
	}
	if s.Fans[0].ID != FanCPU || s.Fans[0].RPM != 2100 || s.Fans[0].Max != 5700 || s.Fans[0].Percent != 37 {
		t.Errorf("cpu fan: %+v", s.Fans[0])
	}
	if s.Fans[1].ID != FanGPU || s.Fans[1].RPM != 1900 || s.Fans[1].Max != 5300 || s.Fans[1].Percent != 36 {
		t.Errorf("gpu fan: %+v", s.Fans[1])
	}
	want := map[string]int{"cpu": 47, "gpu": 43, "sodimm": 44, "other": 45}
	for k, v := range want {
		if s.Temps[k] != v {
			t.Errorf("temp %s: got %d, want %d", k, s.Temps[k], v)
		}
	}
	if s.Profile.Current != "balanced" || len(s.Profile.Choices) != 6 || !s.Profile.Writable {
		t.Errorf("profile: %+v", s.Profile)
	}
	if s.Profile.GmodeForced {
		t.Error("gmodeForced should be false when force_gmode reads N")
	}
	if !s.Turbo.Available || !s.Turbo.Enabled {
		t.Errorf("turbo: %+v", s.Turbo)
	}
	if !s.Power.Available || len(s.Power.Constraints) != 3 {
		t.Fatalf("power: %+v", s.Power)
	}
	if s.Power.Constraints[0].Watts != 65 || s.Power.Constraints[0].Name != "long_term" {
		t.Errorf("constraint 0: %+v", s.Power.Constraints[0])
	}
	if s.Power.Constraints[2].Watts != 215 || s.Power.Constraints[2].Name != "peak_power" {
		t.Errorf("constraint 2: %+v", s.Power.Constraints[2])
	}
	if !s.GPU.Available || s.GPU.Draw == nil || *s.GPU.Draw != 22.4 {
		t.Errorf("gpu: %+v", s.GPU)
	}
	if s.GPU.LimitWritable {
		t.Error("gpu limitWritable must always be false")
	}
	if len(s.Warnings) != 0 {
		t.Errorf("warnings: got %v, want none", s.Warnings)
	}
}

func TestSnapshotEmptyRootDegrades(t *testing.T) {
	r := &Reader{Root: t.TempDir(), RunNvidia: func() (string, error) {
		return "", errors.New("nvidia-smi not found")
	}}
	s := r.Snapshot()
	if len(s.Fans) != 2 {
		t.Fatalf("fans: got %d, want 2 placeholders", len(s.Fans))
	}
	for _, f := range s.Fans {
		if f.RPM != 0 || f.Boost != 0 {
			t.Errorf("fan %s should read zero on a missing tree: %+v", f.ID, f)
		}
	}
	if len(s.Temps) != 0 {
		t.Errorf("temps should be empty, got %v", s.Temps)
	}
	if s.Profile.Writable || s.Turbo.Available || s.Power.Available || s.GPU.Available {
		t.Error("every subsystem should report unavailable on an empty tree")
	}
	if len(s.Warnings) == 0 {
		t.Fatal("want warnings on an empty tree")
	}
	if s.Model != "unknown" {
		t.Errorf("model: got %q, want unknown", s.Model)
	}
}

func TestSnapshotMissingDellSMMOnly(t *testing.T) {
	r := fakeReader(t)
	if err := os.RemoveAll(filepath.Join(r.Root, "sys/class/hwmon/hwmon5")); err != nil {
		t.Fatal(err)
	}
	s := r.Snapshot()
	if _, ok := s.Temps["sodimm"]; ok {
		t.Error("sodimm should be omitted, not zero")
	}
	if _, ok := s.Temps["other"]; ok {
		t.Error("other should be omitted, not zero")
	}
	if s.Temps["cpu"] != 47 {
		t.Errorf("cpu temp should still read: %v", s.Temps)
	}
	found := false
	for _, w := range s.Warnings {
		if w == "dell_smm hwmon not present" {
			found = true
		}
	}
	if !found {
		t.Errorf("want a dell_smm warning, got %v", s.Warnings)
	}
}

func TestSnapshotMissingSingleFile(t *testing.T) {
	r := fakeReader(t)
	if err := os.Remove(filepath.Join(r.Root, "sys/class/hwmon/hwmon4/fan2_input")); err != nil {
		t.Fatal(err)
	}
	s := r.Snapshot()
	if s.Fans[1].RPM != 0 {
		t.Errorf("missing fan2_input should read 0, got %d", s.Fans[1].RPM)
	}
	if len(s.Warnings) == 0 {
		t.Fatal("want a warning for the missing fan input")
	}
}

func TestGmodeForcedTrue(t *testing.T) {
	r := fakeReader(t)
	writeFile(t, r.Root, "sys/module/alienware_wmi/parameters/force_gmode", "Y\n")
	if s := r.Snapshot(); !s.Profile.GmodeForced {
		t.Fatal("gmodeForced should be true when force_gmode reads Y")
	}
}

func TestTurboInverted(t *testing.T) {
	r := fakeReader(t)
	writeFile(t, r.Root, "sys/devices/system/cpu/intel_pstate/no_turbo", "1\n")
	s := r.Snapshot()
	if !s.Turbo.Available || s.Turbo.Enabled {
		t.Fatalf("no_turbo=1 means turbo disabled, got %+v", s.Turbo)
	}
}

func TestPercentClamping(t *testing.T) {
	if got := percentOf(0, 5700); got != 0 {
		t.Errorf("idle: got %d, want 0", got)
	}
	if got := percentOf(9000, 5700); got != 100 {
		t.Errorf("over max: got %d, want 100", got)
	}
	if got := percentOf(2850, 5700); got != 50 {
		t.Errorf("half: got %d, want 50", got)
	}
	if got := percentOf(100, 0); got != 0 {
		t.Errorf("zero max: got %d, want 0", got)
	}
}

func TestParseNvidiaOutputNotAvailableFields(t *testing.T) {
	g, ok := ParseNvidiaOutput("22.40, [N/A], 90.00, [N/A]\n")
	if !ok {
		t.Fatal("want a parsed row")
	}
	if g.Draw == nil || *g.Draw != 22.4 {
		t.Errorf("draw: %+v", g.Draw)
	}
	if g.Limit != nil {
		t.Errorf("limit should be nil, got %v", *g.Limit)
	}
	if g.MaxLimit != nil {
		t.Errorf("maxLimit should be nil, got %v", *g.MaxLimit)
	}
	if g.DefaultLimit == nil || *g.DefaultLimit != 90 {
		t.Errorf("defaultLimit: %+v", g.DefaultLimit)
	}
}

func TestParseNvidiaOutputEmpty(t *testing.T) {
	if _, ok := ParseNvidiaOutput("\n\n"); ok {
		t.Fatal("empty output should not parse")
	}
}

func TestSetBoostWritesSysfs(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetBoost(FanCPU, 128); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := readInt(filepath.Join(r.Root, "sys/class/hwmon/hwmon4/fan1_boost"))
	if err != nil {
		t.Fatal(err)
	}
	if got != 128 {
		t.Fatalf("fan1_boost: got %d, want 128", got)
	}
}

func TestSetBoostRejectsBadInput(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetBoost("middle", 10); !errors.Is(err, ErrBadRequest) {
		t.Errorf("unknown fan: got %v", err)
	}
	if err := r.SetBoost(FanCPU, 999); !errors.Is(err, ErrBadRequest) {
		t.Errorf("out of range: got %v", err)
	}
}

func TestSetBoostMissingHardware(t *testing.T) {
	r := &Reader{Root: t.TempDir()}
	if err := r.SetBoost(FanCPU, 10); !errors.Is(err, ErrHardwareMissing) {
		t.Fatalf("got %v, want ErrHardwareMissing", err)
	}
}

func TestResetBoostZeroesBothFans(t *testing.T) {
	r := fakeReader(t)
	writeFile(t, r.Root, "sys/class/hwmon/hwmon4/fan1_boost", "200\n")
	writeFile(t, r.Root, "sys/class/hwmon/hwmon4/fan2_boost", "180\n")
	if err := r.ResetBoost(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range []string{"fan1_boost", "fan2_boost"} {
		v, err := readInt(filepath.Join(r.Root, "sys/class/hwmon/hwmon4", f))
		if err != nil {
			t.Fatal(err)
		}
		if v != 0 {
			t.Errorf("%s: got %d, want 0", f, v)
		}
	}
}

func TestSetProfileValidatesAgainstChoices(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetProfile("performance"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := readTrimmed(filepath.Join(r.Root, "sys/firmware/acpi/platform_profile"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "performance" {
		t.Fatalf("platform_profile: got %q", got)
	}
	if err := r.SetProfile("turbo-max"); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("got %v, want ErrBadRequest", err)
	}
	if err := r.SetProfile(""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("got %v, want ErrBadRequest", err)
	}
}

func TestSetTurboInvertsWrite(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetTurbo(false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, err := readInt(filepath.Join(r.Root, "sys/devices/system/cpu/intel_pstate/no_turbo"))
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("turbo off should write no_turbo=1, got %d", v)
	}
	if err := r.SetTurbo(true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, _ := readInt(filepath.Join(r.Root, "sys/devices/system/cpu/intel_pstate/no_turbo")); v != 0 {
		t.Fatalf("turbo on should write no_turbo=0, got %d", v)
	}
}

func TestSetTurboUnsupported(t *testing.T) {
	r := &Reader{Root: t.TempDir()}
	if err := r.SetTurbo(true); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("got %v, want ErrNotSupported", err)
	}
}

func TestSetPowerLimitConvertsToMicrowatts(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetPowerLimit(0, 45); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, err := readInt(filepath.Join(r.Root, "sys/class/powercap/intel-rapl:0/constraint_0_power_limit_uw"))
	if err != nil {
		t.Fatal(err)
	}
	if v != 45000000 {
		t.Fatalf("got %d microwatts, want 45000000", v)
	}
}

func TestSetPowerLimitRejects(t *testing.T) {
	r := fakeReader(t)
	if err := r.SetPowerLimit(9, 45); !errors.Is(err, ErrBadRequest) {
		t.Errorf("bad constraint: got %v", err)
	}
	if err := r.SetPowerLimit(0, 0); !errors.Is(err, ErrBadRequest) {
		t.Errorf("bad watts: got %v", err)
	}
	empty := &Reader{Root: t.TempDir()}
	if err := empty.SetPowerLimit(0, 45); !errors.Is(err, ErrHardwareMissing) {
		t.Errorf("missing constraint: got %v", err)
	}
}

func TestSetPowerLimitReadOnlyConstraint(t *testing.T) {
	r := fakeReader(t)
	path := filepath.Join(r.Root, "sys/class/powercap/intel-rapl:0/constraint_1_power_limit_uw")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file mode permission checks")
	}
	if probeWritable(path) {
		t.Fatal("probeWritable should report false for a read only file")
	}
	if err := r.SetPowerLimit(1, 100); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("got %v, want ErrNotSupported", err)
	}
	s := r.Snapshot()
	if s.Power.Constraints[1].Writable {
		t.Fatal("constraint 1 should report writable false")
	}
	if v, _ := readInt(path); v != 140000000 {
		t.Fatalf("probing must not change the value, got %d", v)
	}
}

func TestPPDHook(t *testing.T) {
	r := fakeReader(t)
	r.PPDRunning = func() bool { return true }
	if !r.Snapshot().Profile.PPDRunning {
		t.Fatal("ppdRunning should follow the hook")
	}
}

func TestReadCPUTakesHighestCore(t *testing.T) {
	r := fakeReader(t)
	s := r.Snapshot()
	if !s.CPU.Available {
		t.Fatalf("cpu should be available, got %+v", s.CPU)
	}
	if s.CPU.MHz != 4700 {
		t.Fatalf("want the highest core 4700 MHz, got %d", s.CPU.MHz)
	}
}

func TestReadCPUMissingDegrades(t *testing.T) {
	r := &Reader{Root: t.TempDir()}
	s := r.Snapshot()
	if s.CPU.Available || s.CPU.MHz != 0 {
		t.Fatalf("absent cpufreq must report unavailable, got %+v", s.CPU)
	}
	found := false
	for _, warn := range s.Warnings {
		if warn == "cpufreq scaling_cur_freq not present" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing cpufreq should warn, got %v", s.Warnings)
	}
}

func TestReadCPUUnreadableValuesDegrade(t *testing.T) {
	root := fakeRoot(t)
	writeFile(t, root, "sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq", "\n")
	writeFile(t, root, "sys/devices/system/cpu/cpu1/cpufreq/scaling_cur_freq", "not-a-number\n")
	r := &Reader{Root: root}
	s := r.Snapshot()
	if s.CPU.Available {
		t.Fatalf("garbage cpufreq must not report available, got %+v", s.CPU)
	}
}
