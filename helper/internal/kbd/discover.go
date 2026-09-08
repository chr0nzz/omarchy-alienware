package kbd

import (
	"os"
	"strings"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

const DeviceEnv = "ALIENWARECTL_KBD_DEVICE"

var findDevice = hidraw.FindOneByVIDPID

func discover() (hidraw.DevInfo, error) {
	info, err := findDevice(DefaultVendorID, DefaultProductID)
	if err != nil {
		return hidraw.DevInfo{}, newError(CodeNoDevice, "%s", err.Error())
	}
	return info, nil
}

func devicePath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(DeviceEnv)); p != "" {
		return p, nil
	}
	info, err := discover()
	if err != nil {
		return "", err
	}
	return info.Path, nil
}

func OpenDefault() (*Device, error) {
	path, err := devicePath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

func Present() bool {
	_, err := discover()
	return err == nil
}

func KeyCount() int {
	return DefaultKeyLast - DefaultKeyFirst + 1
}
