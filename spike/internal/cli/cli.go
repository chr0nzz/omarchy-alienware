package cli

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/elc"
	"github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/hidraw"
)

var Version = "0.0.1-spike"

const Usage = `afxspike <verb> [args] [--device=/dev/hidrawN] [--pid=0550] [--transport=output|write|feature]

  probe                  open the device and print its status and raw config as JSON
  sweep <first> <last>   light hardware zone IDs first..last one at a time
                         flags: --dwell=2s
  set <zoneid> <RRGGBB>  light a single hardware zone ID
  all <RRGGBB>           light the default zone sweep (0-31)
  off                    blank the default zone sweep (0-31)
  version                print the spike CLI version

CRITICAL: stop openrgb-server before running this. Two processes touching the
AW-ELC HID node at once wedges the controller's firmware. See README.md.
`

const (
	defaultZoneFirst = 0
	defaultZoneLast  = 31
)

type Env struct {
	Stdout io.Writer
	Stderr io.Writer
}

func Run(args []string) int {
	return RunWith(Env{Stdout: os.Stdout, Stderr: os.Stderr}, args)
}

func RunWith(env Env, args []string) int {
	positional, flags := splitFlags(args)
	if len(positional) == 0 {
		io.WriteString(env.Stderr, Usage)
		return emitError(env.Stdout, badRequest("a verb is required"))
	}

	switch positional[0] {
	case "version", "--version", "-v":
		return emitOK(env.Stdout, struct {
			OK      bool   `json:"ok"`
			Version string `json:"version"`
		}{true, Version})
	case "help", "--help", "-h":
		io.WriteString(env.Stdout, Usage)
		return 0
	case "probe":
		return runProbe(env, flags)
	case "sweep":
		return runSweep(env, positional[1:], flags)
	case "set":
		return runSet(env, positional[1:], flags)
	case "all":
		return runAll(env, positional[1:], flags)
	case "zones":
		return runZones(env, positional[1:], flags)
	case "off":
		return runOff(env, flags)
	}
	io.WriteString(env.Stderr, Usage)
	return emitError(env.Stdout, badRequest("unknown verb %q", positional[0]))
}

func splitFlags(args []string) ([]string, map[string]string) {
	positional := make([]string, 0, len(args))
	flags := map[string]string{}
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			kv := strings.SplitN(strings.TrimPrefix(a, "--"), "=", 2)
			if len(kv) == 2 {
				flags[kv[0]] = kv[1]
			} else {
				flags[kv[0]] = ""
			}
			continue
		}
		positional = append(positional, a)
	}
	return positional, flags
}

func openDevice(flags map[string]string) (*elc.Device, error) {
	writeMode, err := elc.ParseWriteMode(flags["transport"])
	if err != nil {
		return nil, badRequest("%s", err.Error())
	}

	path := flags["device"]
	if path == "" {
		pid := elc.DefaultProductID
		if v, ok := flags["pid"]; ok {
			parsed, perr := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 16)
			if perr != nil {
				return nil, badRequest("pid %q must be hex, for example 0550", v)
			}
			pid = uint16(parsed)
		}
		info, ferr := hidraw.FindOneByVIDPID(elc.DefaultVendorID, pid)
		if ferr != nil {
			return nil, internalf("could not find a hidraw node for vid=0x%04x pid=0x%04x: %s", elc.DefaultVendorID, pid, ferr.Error())
		}
		path = info.Path
	}

	dev, oerr := elc.Open(path, writeMode)
	if oerr != nil {
		return nil, internalf("could not open %s: %s", path, oerr.Error())
	}
	return dev, nil
}

func defaultZoneRange() []uint8 {
	zones := make([]uint8, 0, defaultZoneLast-defaultZoneFirst+1)
	for i := defaultZoneFirst; i <= defaultZoneLast; i++ {
		zones = append(zones, uint8(i))
	}
	return zones
}

type probeStatus struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	RawHex string `json:"rawHex,omitempty"`
	Status uint8  `json:"statusByte,omitempty"`
	Name   string `json:"name,omitempty"`
}

type probeFeature struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	RawHex string `json:"rawHex,omitempty"`
}

func runProbe(env Env, flags map[string]string) int {
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	info, _ := dev.RawInfo()

	status := probeStatus{}
	allZero := true
	if st, serr := dev.Status(); serr != nil {
		status.Error = serr.Error()
	} else {
		status.OK = true
		status.RawHex = hex.EncodeToString(st.Raw)
		status.Status = st.Status
		status.Name = st.Name
		for _, b := range st.Raw {
			if b != 0 {
				allZero = false
				break
			}
		}
	}

	feature := probeFeature{}
	if fb, ferr := dev.FeatureProbe(); ferr != nil {
		feature.Error = ferr.Error()
	} else {
		feature.OK = true
		feature.RawHex = hex.EncodeToString(fb)
	}

	looksWedged := status.OK && allZero

	return emitOK(env.Stdout, struct {
		OK          bool         `json:"ok"`
		Path        string       `json:"path"`
		VendorID    string       `json:"vendorId"`
		ProductID   string       `json:"productId"`
		BusType     uint32       `json:"busType"`
		Status      probeStatus  `json:"status"`
		Feature     probeFeature `json:"feature"`
		LooksWedged bool         `json:"looksWedged"`
	}{true, dev.Path(), fmt.Sprintf("0x%04x", uint16(info.Vendor)), fmt.Sprintf("0x%04x", uint16(info.Product)), info.BusType, status, feature, looksWedged})
}

func runSweep(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("sweep takes a first and last zone id, for example sweep 0 19"))
	}
	first, err1 := strconv.Atoi(args[0])
	last, err2 := strconv.Atoi(args[1])
	if err1 != nil || err2 != nil || first < 0 || first > 255 || last < 0 || last > 255 {
		return emitError(env.Stdout, badRequest("first and last must be zone ids 0-255"))
	}

	dwell := 2 * time.Second
	if v, ok := flags["dwell"]; ok {
		d, derr := time.ParseDuration(v)
		if derr != nil || d <= 0 {
			return emitError(env.Stdout, badRequest("dwell %q must be a positive duration, for example 2s", v))
		}
		dwell = d
	}

	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	step := 1
	if last < first {
		step = -1
	}
	for id := first; ; id += step {
		if serr := dev.SetZoneColor(255, 255, 255, []uint8{uint8(id)}); serr != nil {
			return emitError(env.Stdout, internalf("zone %d: %s", id, serr.Error()))
		}
		emitOK(env.Stdout, struct {
			OK      bool  `json:"ok"`
			Zone    int   `json:"zone"`
			DwellMs int64 `json:"dwellMs"`
		}{true, id, dwell.Milliseconds()})
		if id == last {
			break
		}
		time.Sleep(dwell)
	}
	return 0
}

func runSet(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("set takes a zone id and a colour, for example set 4 ff8800"))
	}
	zone, zerr := strconv.Atoi(args[0])
	if zerr != nil || zone < 0 || zone > 255 {
		return emitError(env.Stdout, badRequest("zone id %q must be 0-255", args[0]))
	}
	r, g, b, cerr := parseHexColor(args[1])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}

	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	if serr := dev.SetZoneColor(r, g, b, []uint8{uint8(zone)}); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK    bool   `json:"ok"`
		Zone  int    `json:"zone"`
		Color string `json:"color"`
	}{true, zone, hexColor(r, g, b)})
}

func runZones(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("zones takes a comma separated id list and a colour, for example zones 0,1,4 ff8800"))
	}
	fields := strings.Split(args[0], ",")
	ids := make([]uint8, 0, len(fields))
	for _, f := range fields {
		t := strings.TrimSpace(f)
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return emitError(env.Stdout, badRequest("zone id %q must be 0-255", t))
		}
		ids = append(ids, uint8(n))
	}
	if len(ids) == 0 {
		return emitError(env.Stdout, badRequest("zones needs at least one id"))
	}
	r, g, b, cerr := parseHexColor(args[1])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()
	if serr := dev.SetZoneColor(r, g, b, ids); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK    bool    `json:"ok"`
		Zones []uint8 `json:"zones"`
		Color string  `json:"color"`
	}{true, ids, hexColor(r, g, b)})
}

func runAll(env Env, args []string, flags map[string]string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("all takes a colour, for example all ff8800"))
	}
	r, g, b, cerr := parseHexColor(args[0])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}

	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	if serr := dev.SetZoneColor(r, g, b, defaultZoneRange()); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK        bool   `json:"ok"`
		Color     string `json:"color"`
		ZoneFirst int    `json:"zoneFirst"`
		ZoneLast  int    `json:"zoneLast"`
	}{true, hexColor(r, g, b), defaultZoneFirst, defaultZoneLast})
}

func runOff(env Env, flags map[string]string) int {
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	if serr := dev.SetZoneColor(0, 0, 0, defaultZoneRange()); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK bool `json:"ok"`
	}{true})
}
