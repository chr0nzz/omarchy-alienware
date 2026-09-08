package hw

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const nvidiaTimeout = 2 * time.Second

type fanSpec struct {
	id       string
	index    int
	label    string
	fallback int
}

var fanSpecs = []fanSpec{
	{id: FanCPU, index: 1, label: "CPU Fan", fallback: defaultCPUFanMax},
	{id: FanGPU, index: 2, label: "GPU Fan", fallback: defaultGPUFanMax},
}

var constraintNames = []string{"long_term", "short_term", "peak_power"}

func percentOf(rpm, max int) int {
	if max <= 0 {
		return 0
	}
	p := int(math.Round(float64(rpm) / float64(max) * 100))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func (r *Reader) readFans(hwmon string, w *warnings) []Fan {
	fans := make([]Fan, 0, len(fanSpecs))
	for _, spec := range fanSpecs {
		f := Fan{ID: spec.id, Index: spec.index, Label: spec.label, Max: spec.fallback}
		if hwmon == "" {
			w.add("fan%d not readable: alienware_wmi hwmon not present", spec.index)
			fans = append(fans, f)
			continue
		}
		if label, err := readTrimmed(filepath.Join(hwmon, fmt.Sprintf("fan%d_label", spec.index))); err == nil && label != "" {
			f.Label = label
		}
		if max, err := readInt(filepath.Join(hwmon, fmt.Sprintf("fan%d_max", spec.index))); err == nil && max > 0 {
			f.Max = max
		}
		if rpm, err := readInt(filepath.Join(hwmon, fmt.Sprintf("fan%d_input", spec.index))); err == nil {
			f.RPM = rpm
		} else {
			w.add("fan%d_input not readable", spec.index)
		}
		if boost, err := readInt(filepath.Join(hwmon, fmt.Sprintf("fan%d_boost", spec.index))); err == nil {
			f.Boost = boost
		} else {
			w.add("fan%d_boost not readable", spec.index)
		}
		f.Percent = percentOf(f.RPM, f.Max)
		fans = append(fans, f)
	}
	return fans
}

func (r *Reader) readTemps(alienware, dellsmm string, w *warnings) map[string]int {
	temps := map[string]int{}
	type src struct {
		key  string
		dir  string
		file string
		miss string
	}
	sources := []src{
		{key: "cpu", dir: alienware, file: "temp1_input", miss: "alienware_wmi hwmon not present"},
		{key: "gpu", dir: alienware, file: "temp2_input", miss: "alienware_wmi hwmon not present"},
		{key: "sodimm", dir: dellsmm, file: "temp3_input", miss: "dell_smm hwmon not present"},
		{key: "other", dir: dellsmm, file: "temp6_input", miss: "dell_smm hwmon not present"},
	}
	for _, s := range sources {
		if s.dir == "" {
			w.add("%s", s.miss)
			continue
		}
		milli, err := readInt(filepath.Join(s.dir, s.file))
		if err != nil {
			w.add("%s not readable in %s", s.file, filepath.Base(s.dir))
			continue
		}
		temps[s.key] = int(math.Round(float64(milli) / 1000.0))
	}
	return temps
}

func (r *Reader) profilePath() string {
	return r.path("sys", "firmware", "acpi", "platform_profile")
}

func (r *Reader) profileChoicesPath() string {
	return r.path("sys", "firmware", "acpi", "platform_profile_choices")
}

func (r *Reader) forceGmodePath() string {
	return r.path("sys", "module", "alienware_wmi", "parameters", "force_gmode")
}

func (r *Reader) ReadProfile() (string, error) {
	return readTrimmed(r.profilePath())
}

func (r *Reader) ReadProfileChoices() ([]string, error) {
	s, err := readTrimmed(r.profileChoicesPath())
	if err != nil {
		return nil, err
	}
	return strings.Fields(s), nil
}

func (r *Reader) readProfile(w *warnings) Profile {
	p := Profile{Choices: []string{}}
	current, err := r.ReadProfile()
	if err != nil {
		w.add("platform_profile not readable")
	} else {
		p.Current = current
		p.Writable = probeWritable(r.profilePath())
	}
	choices, err := r.ReadProfileChoices()
	if err != nil {
		w.add("platform_profile_choices not readable")
	} else {
		p.Choices = choices
	}
	if g, err := readTrimmed(r.forceGmodePath()); err == nil {
		switch strings.ToUpper(strings.TrimSpace(g)) {
		case "Y", "1", "YES", "TRUE":
			p.GmodeForced = true
		}
	}
	if r.PPDRunning != nil {
		p.PPDRunning = r.PPDRunning()
	}
	return p
}

func (r *Reader) noTurboPath() string {
	return r.path("sys", "devices", "system", "cpu", "intel_pstate", "no_turbo")
}

func (r *Reader) cpufreqGlob() string {
	return filepath.Join(r.path("sys", "devices", "system", "cpu"), "cpu[0-9]*", "cpufreq", "scaling_cur_freq")
}

func (r *Reader) readCPU(w *warnings) CPU {
	matches, err := filepath.Glob(r.cpufreqGlob())
	if err != nil || len(matches) == 0 {
		w.add("cpufreq scaling_cur_freq not present")
		return CPU{}
	}
	best := 0
	found := false
	for _, m := range matches {
		khz, err := readInt(m)
		if err != nil || khz <= 0 {
			continue
		}
		found = true
		if khz > best {
			best = khz
		}
	}
	if !found {
		w.add("cpufreq scaling_cur_freq unreadable")
		return CPU{}
	}
	return CPU{Available: true, MHz: (best + 500) / 1000}
}

func (r *Reader) readTurbo(w *warnings) Turbo {
	v, err := readInt(r.noTurboPath())
	if err != nil {
		w.add("intel_pstate no_turbo not present")
		return Turbo{}
	}
	return Turbo{Available: true, Enabled: v == 0}
}

func (r *Reader) raplDir() string {
	return r.path("sys", "class", "powercap", "intel-rapl:0")
}

func (r *Reader) constraintPath(index int) string {
	return filepath.Join(r.raplDir(), fmt.Sprintf("constraint_%d_power_limit_uw", index))
}

func (r *Reader) readPower(w *warnings) Power {
	p := Power{Constraints: []Constraint{}}
	if _, err := os.Stat(r.raplDir()); err != nil {
		w.add("intel-rapl powercap not present")
		return p
	}
	for i, fallbackName := range constraintNames {
		path := r.constraintPath(i)
		uw, err := readInt(path)
		if err != nil {
			continue
		}
		name := fallbackName
		if n, err := readTrimmed(filepath.Join(r.raplDir(), fmt.Sprintf("constraint_%d_name", i))); err == nil && n != "" {
			name = n
		}
		p.Constraints = append(p.Constraints, Constraint{
			Index:    i,
			Name:     name,
			Watts:    int(math.Round(float64(uw) / 1000000.0)),
			Writable: probeWritable(path),
		})
	}
	if len(p.Constraints) == 0 {
		w.add("no readable intel-rapl power constraints")
		return p
	}
	p.Available = true
	return p
}

func parseNvidiaField(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "[") {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func (r *Reader) runNvidia() (string, error) {
	if r.RunNvidia != nil {
		return r.RunNvidia()
	}
	bin, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return "", fmt.Errorf("nvidia-smi not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), nvidiaTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin,
		"--query-gpu=power.draw,power.limit,power.default_limit,power.max_limit",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func ParseNvidiaOutput(out string) (GPU, bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		return GPU{
			Available:     true,
			Draw:          parseNvidiaField(parts[0]),
			Limit:         parseNvidiaField(parts[1]),
			DefaultLimit:  parseNvidiaField(parts[2]),
			MaxLimit:      parseNvidiaField(parts[3]),
			LimitWritable: false,
		}, true
	}
	return GPU{}, false
}

func (r *Reader) readGPU(w *warnings) GPU {
	out, err := r.runNvidia()
	if err != nil {
		w.add("nvidia-smi unavailable")
		return GPU{}
	}
	g, ok := ParseNvidiaOutput(out)
	if !ok {
		w.add("nvidia-smi returned no usable rows")
		return GPU{}
	}
	return g
}

func (r *Reader) GPU() GPU {
	w := &warnings{}
	return r.readGPU(w)
}

func (r *Reader) Snapshot() Snapshot {
	w := &warnings{}
	alienware, err := r.FindHwmon(AlienwareHwmonName)
	if err != nil {
		alienware = ""
		w.add("alienware_wmi hwmon not present")
	}
	dellsmm, err := r.FindHwmon(DellSMMHwmonName)
	if err != nil {
		dellsmm = ""
	}
	s := Snapshot{
		Model:   r.Model(),
		Hwmon:   alienware,
		Fans:    r.readFans(alienware, w),
		Temps:   r.readTemps(alienware, dellsmm, w),
		Profile: r.readProfile(w),
		Turbo:   r.readTurbo(w),
		Power:   r.readPower(w),
		CPU:     r.readCPU(w),
		GPU:     r.readGPU(w),
	}
	s.Warnings = w.result()
	return s
}

func (r *Reader) Temps() (map[string]int, []string) {
	w := &warnings{}
	alienware, err := r.FindHwmon(AlienwareHwmonName)
	if err != nil {
		alienware = ""
	}
	dellsmm, err := r.FindHwmon(DellSMMHwmonName)
	if err != nil {
		dellsmm = ""
	}
	return r.readTemps(alienware, dellsmm, w), w.result()
}
