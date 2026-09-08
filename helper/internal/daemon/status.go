package daemon

import (
	"encoding/json"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
)

type CurveStatus struct {
	Active     bool        `json:"active"`
	Interval   int         `json:"interval"`
	Hysteresis int         `json:"hysteresis"`
	CPU        []fan.Point `json:"cpu"`
	GPU        []fan.Point `json:"gpu"`
}

type Status struct {
	OK       bool           `json:"ok"`
	TS       int64          `json:"ts"`
	Model    string         `json:"model"`
	Hwmon    string         `json:"hwmon"`
	Fans     []hw.Fan       `json:"fans"`
	Temps    map[string]int `json:"temps"`
	Profile  hw.Profile     `json:"profile"`
	Turbo    hw.Turbo       `json:"turbo"`
	Power    hw.Power       `json:"power"`
	CPU      hw.CPU         `json:"cpu"`
	GPU      hw.GPU         `json:"gpu"`
	Curve    CurveStatus    `json:"curve"`
	Warnings []string       `json:"warnings"`
}

func BuildStatus(snap hw.Snapshot, curve CurveStatus) Status {
	if snap.Temps == nil {
		snap.Temps = map[string]int{}
	}
	if snap.Fans == nil {
		snap.Fans = []hw.Fan{}
	}
	if snap.Warnings == nil {
		snap.Warnings = []string{}
	}
	if curve.CPU == nil {
		curve.CPU = []fan.Point{}
	}
	if curve.GPU == nil {
		curve.GPU = []fan.Point{}
	}
	return Status{
		OK:       true,
		TS:       time.Now().Unix(),
		Model:    snap.Model,
		Hwmon:    snap.Hwmon,
		Fans:     snap.Fans,
		Temps:    snap.Temps,
		Profile:  snap.Profile,
		Turbo:    snap.Turbo,
		Power:    snap.Power,
		CPU:      snap.CPU,
		GPU:      snap.GPU,
		Curve:    curve,
		Warnings: snap.Warnings,
	}
}

func (s Status) JSON() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
