package hw

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"syscall"
)

var (
	ErrHardwareMissing = errors.New("hardware path not present")
	ErrBadRequest      = errors.New("bad request")
	ErrNotSupported    = errors.New("not supported")
)

func classifyWrite(path string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrHardwareMissing, err.Error())
	}
	if errors.Is(err, fs.ErrPermission) ||
		errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.ENODEV) {
		return fmt.Errorf("%w: %s refused the write: %s", ErrNotSupported, path, err.Error())
	}
	return fmt.Errorf("%s refused the write: %w", path, err)
}

func fanIndex(id string) (int, error) {
	for _, spec := range fanSpecs {
		if spec.id == id {
			return spec.index, nil
		}
	}
	return 0, fmt.Errorf("%w: unknown fan %q, expected cpu or gpu", ErrBadRequest, id)
}

func (r *Reader) boostPath(hwmon string, index int) string {
	return filepath.Join(hwmon, fmt.Sprintf("fan%d_boost", index))
}

func (r *Reader) SetBoost(id string, value int) error {
	index, err := fanIndex(id)
	if err != nil {
		return err
	}
	if value < 0 || value > 255 {
		return fmt.Errorf("%w: boost %d out of range 0-255", ErrBadRequest, value)
	}
	hwmon, err := r.FindHwmon(AlienwareHwmonName)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrHardwareMissing, err.Error())
	}
	path := r.boostPath(hwmon, index)
	if err := writeSysfs(path, strconv.Itoa(value)); err != nil {
		return classifyWrite(path, err)
	}
	return nil
}

func (r *Reader) ResetBoost() error {
	hwmon, err := r.FindHwmon(AlienwareHwmonName)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrHardwareMissing, err.Error())
	}
	var firstErr error
	for _, spec := range fanSpecs {
		if err := writeSysfs(r.boostPath(hwmon, spec.index), "0"); err != nil && firstErr == nil {
			firstErr = classifyWrite(r.boostPath(hwmon, spec.index), err)
		}
	}
	return firstErr
}

func (r *Reader) SetProfile(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty profile name", ErrBadRequest)
	}
	choices, err := r.ReadProfileChoices()
	if err != nil {
		return fmt.Errorf("%w: platform_profile_choices not readable", ErrHardwareMissing)
	}
	ok := false
	for _, c := range choices {
		if c == name {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("%w: profile %q is not one of %v", ErrBadRequest, name, choices)
	}
	if err := writeSysfs(r.profilePath(), name); err != nil {
		return classifyWrite(r.profilePath(), err)
	}
	return nil
}

func (r *Reader) SetTurbo(on bool) error {
	value := "1"
	if on {
		value = "0"
	}
	path := r.noTurboPath()
	if _, err := readInt(path); err != nil {
		return fmt.Errorf("%w: intel_pstate no_turbo not present", ErrNotSupported)
	}
	if err := writeSysfs(path, value); err != nil {
		return classifyWrite(path, err)
	}
	return nil
}

func (r *Reader) SetPowerLimit(index, watts int) error {
	if index < 0 || index >= len(constraintNames) {
		return fmt.Errorf("%w: constraint %d out of range 0-%d", ErrBadRequest, index, len(constraintNames)-1)
	}
	if watts < 1 || watts > 1000 {
		return fmt.Errorf("%w: %d watts out of range 1-1000", ErrBadRequest, watts)
	}
	path := r.constraintPath(index)
	if _, err := readInt(path); err != nil {
		return fmt.Errorf("%w: constraint %d not present", ErrHardwareMissing, index)
	}
	if !probeWritable(path) {
		return fmt.Errorf("%w: constraint %d is read only on this firmware", ErrNotSupported, index)
	}
	if err := writeSysfs(path, strconv.Itoa(watts*1000000)); err != nil {
		return classifyWrite(path, err)
	}
	return nil
}
