package hw

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	AlienwareHwmonName = "alienware_wmi"
	DellSMMHwmonName   = "dell_smm"

	FanCPU = "cpu"
	FanGPU = "gpu"

	defaultCPUFanMax = 5700
	defaultGPUFanMax = 5300
)

type Fan struct {
	ID      string `json:"id"`
	Index   int    `json:"index"`
	Label   string `json:"label"`
	RPM     int    `json:"rpm"`
	Max     int    `json:"max"`
	Boost   int    `json:"boost"`
	Percent int    `json:"percent"`
}

type Profile struct {
	Current     string   `json:"current"`
	Choices     []string `json:"choices"`
	PPDRunning  bool     `json:"ppdRunning"`
	GmodeForced bool     `json:"gmodeForced"`
	Writable    bool     `json:"writable"`
}

type Turbo struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

type Constraint struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Watts    int    `json:"watts"`
	Writable bool   `json:"writable"`
}

type Power struct {
	Available   bool         `json:"available"`
	Constraints []Constraint `json:"constraints"`
}

type GPU struct {
	Available     bool     `json:"available"`
	Draw          *float64 `json:"draw"`
	Limit         *float64 `json:"limit"`
	DefaultLimit  *float64 `json:"defaultLimit"`
	MaxLimit      *float64 `json:"maxLimit"`
	LimitWritable bool     `json:"limitWritable"`
}

type CPU struct {
	Available bool `json:"available"`
	MHz       int  `json:"mhz"`
}

type Snapshot struct {
	Model    string
	Hwmon    string
	Fans     []Fan
	Temps    map[string]int
	Profile  Profile
	Turbo    Turbo
	Power    Power
	CPU      CPU
	GPU      GPU
	Warnings []string
}

type Reader struct {
	Root       string
	PPDRunning func() bool
	RunNvidia  func() (string, error)
}

func New() *Reader {
	return &Reader{Root: "/"}
}

func (r *Reader) root() string {
	if r.Root == "" {
		return "/"
	}
	return r.Root
}

func (r *Reader) path(parts ...string) string {
	return filepath.Join(append([]string{r.root()}, parts...)...)
}

type warnings struct {
	list []string
	seen map[string]bool
}

func (w *warnings) add(format string, args ...any) {
	if w.seen == nil {
		w.seen = map[string]bool{}
	}
	msg := fmt.Sprintf(format, args...)
	if w.seen[msg] {
		return
	}
	w.seen[msg] = true
	w.list = append(w.list, msg)
}

func (w *warnings) result() []string {
	if w.list == nil {
		return []string{}
	}
	return w.list
}

func readTrimmed(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func readInt(path string) (int, error) {
	s, err := readTrimmed(path)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return 0, fmt.Errorf("%s is empty", path)
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", path, err)
	}
	return int(v), nil
}

func writeSysfs(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(value); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func probeWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func (r *Reader) FindHwmon(name string) (string, error) {
	base := r.path("sys", "class", "hwmon")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("cannot list %s: %w", base, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		dir := filepath.Join(base, n)
		got, err := readTrimmed(filepath.Join(dir, "name"))
		if err != nil {
			continue
		}
		if got == name {
			return dir, nil
		}
	}
	return "", fmt.Errorf("%s hwmon not present", name)
}

func (r *Reader) Model() string {
	for _, p := range []string{"product_name", "board_name"} {
		if s, err := readTrimmed(r.path("sys", "class", "dmi", "id", p)); err == nil && s != "" {
			return s
		}
	}
	return "unknown"
}
