package kbdcli

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/hidraw"
	"github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/kbd"
)

const Usage = `kbdspike <verb> [args] [--device=/dev/hidrawN]

  probe                     open the device and print protocol and status info as JSON
  all <RRGGBB>              set every key to one colour
  key <index> <RRGGBB>      set a single key by its protocol index
  sweep <first> <last>      light key indices one at a time, flags: --dwell=2s
  off                       set every key to black

CRITICAL: this device (0d62:babc) is normally root-only by design and is the
keyboard's own input node. Only run this as root, only when you mean to write
to hardware, and never point --device at anything you have not confirmed is
this controller.
`

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
	case "help", "--help", "-h":
		io.WriteString(env.Stdout, Usage)
		return 0
	case "probe":
		return runProbe(env, flags)
	case "all":
		return runAll(env, positional[1:], flags)
	case "keys":
		return runKeys(env, positional[1:], flags)
	case "key":
		return runKey(env, positional[1:], flags)
	case "sweep":
		return runSweep(env, positional[1:], flags)
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

func openDevice(flags map[string]string) (*kbd.Device, error) {
	path := flags["device"]
	if path == "" {
		info, ferr := hidraw.FindOneByVIDPID(kbd.DefaultVendorID, kbd.DefaultProductID)
		if ferr != nil {
			return nil, internalf("could not find a hidraw node for vid=0x%04x pid=0x%04x: %s", kbd.DefaultVendorID, kbd.DefaultProductID, ferr.Error())
		}
		path = info.Path
	}

	dev, oerr := kbd.Open(path)
	if oerr != nil {
		return nil, internalf("could not open %s: %s", path, oerr.Error())
	}
	return dev, nil
}

type probeStatus struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	RawHex string `json:"rawHex,omitempty"`
	Status uint8  `json:"statusByte,omitempty"`
	Name   string `json:"name,omitempty"`
	Ready  bool   `json:"ready,omitempty"`
}

func runProbe(env Env, flags map[string]string) int {
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	info, _ := dev.RawInfo()

	status := probeStatus{}
	if st, serr := dev.Status(); serr != nil {
		status.Error = serr.Error()
	} else {
		status.OK = true
		status.RawHex = hex.EncodeToString(st.Raw)
		status.Status = st.Status
		status.Name = st.Name
		status.Ready = st.Ready
	}

	return emitOK(env.Stdout, struct {
		OK                bool        `json:"ok"`
		Path              string      `json:"path"`
		VendorID          string      `json:"vendorId"`
		ProductID         string      `json:"productId"`
		BusType           uint32      `json:"busType"`
		ReportLength      int         `json:"reportLength"`
		FeatureReportID   string      `json:"featureReportId"`
		MaxKeysPerFrame   int         `json:"maxKeysPerColorSetFrame"`
		KeyTableAvailable bool        `json:"keyTableAvailable"`
		KeyIndexNote      string      `json:"keyIndexNote"`
		DefaultKeyFirst   int         `json:"defaultKeyFirst"`
		DefaultKeyLast    int         `json:"defaultKeyLast"`
		Status            probeStatus `json:"status"`
	}{
		true,
		dev.Path(),
		fmt.Sprintf("0x%04x", uint16(info.Vendor)),
		fmt.Sprintf("0x%04x", uint16(info.Product)),
		info.BusType,
		kbd.ReportLength,
		fmt.Sprintf("0x%02x", kbd.FeatureReportID),
		kbd.MaxKeysPerColorSetFrame,
		kbd.KeyTableAvailable,
		kbd.KeyIndexNote,
		kbd.DefaultKeyFirst,
		kbd.DefaultKeyLast,
		status,
	})
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

	if serr := dev.SetAllKeys(r, g, b, kbd.DefaultKeyFirst, kbd.DefaultKeyLast); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK       bool   `json:"ok"`
		Color    string `json:"color"`
		KeyFirst int    `json:"keyFirst"`
		KeyLast  int    `json:"keyLast"`
	}{true, hexColor(r, g, b), kbd.DefaultKeyFirst, kbd.DefaultKeyLast})
}

func runKeys(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("keys takes a comma separated index list and a colour, for example keys 0,1,2 ff8800"))
	}
	r, g, b, cerr := parseHexColor(args[1])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}
	fields := strings.Split(args[0], ",")
	wanted := make([]kbd.KeyColor, 0, len(fields))
	indexes := make([]int, 0, len(fields))
	for _, f := range fields {
		t := strings.TrimSpace(f)
		if t == "" {
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return emitError(env.Stdout, badRequest("key index %q must be 0-255", t))
		}
		wanted = append(wanted, kbd.KeyColor{Index: uint8(n), R: r, G: g, B: b})
		indexes = append(indexes, n)
	}
	if len(wanted) == 0 {
		return emitError(env.Stdout, badRequest("keys needs at least one index"))
	}
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()
	if serr := dev.SetKeyColors(wanted); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK    bool   `json:"ok"`
		Keys  []int  `json:"keys"`
		Color string `json:"color"`
	}{true, indexes, hexColor(r, g, b)})
}

func runKey(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("key takes an index and a colour, for example key 4 ff8800"))
	}
	index, ierr := strconv.Atoi(args[0])
	if ierr != nil || index < 0 || index > 255 {
		return emitError(env.Stdout, badRequest("key index %q must be 0-255", args[0]))
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

	if serr := dev.SetKeyColors([]kbd.KeyColor{{Index: uint8(index), R: r, G: g, B: b}}); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	name, hasName := kbd.KeyName(uint8(index))
	return emitOK(env.Stdout, struct {
		OK    bool   `json:"ok"`
		Key   int    `json:"key"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color"`
	}{true, index, nameOrEmpty(name, hasName), hexColor(r, g, b)})
}

func nameOrEmpty(name string, ok bool) string {
	if !ok {
		return ""
	}
	return name
}

func runSweep(env Env, args []string, flags map[string]string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("sweep takes a first and last key index, for example sweep 0 135"))
	}
	first, err1 := strconv.Atoi(args[0])
	last, err2 := strconv.Atoi(args[1])
	if err1 != nil || err2 != nil || first < 0 || first > 255 || last < 0 || last > 255 {
		return emitError(env.Stdout, badRequest("first and last must be key indices 0-255"))
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
		if serr := dev.SetKeyColors([]kbd.KeyColor{{Index: uint8(id), R: 255, G: 255, B: 255}}); serr != nil {
			return emitError(env.Stdout, internalf("key %d: %s", id, serr.Error()))
		}
		name, hasName := kbd.KeyName(uint8(id))
		emitOK(env.Stdout, struct {
			OK      bool   `json:"ok"`
			Key     int    `json:"key"`
			Name    string `json:"name,omitempty"`
			DwellMs int64  `json:"dwellMs"`
		}{true, id, nameOrEmpty(name, hasName), dwell.Milliseconds()})
		if id == last {
			break
		}
		time.Sleep(dwell)
	}
	return 0
}

func runOff(env Env, flags map[string]string) int {
	dev, err := openDevice(flags)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer dev.Close()

	if serr := dev.SetAllKeys(0, 0, 0, kbd.DefaultKeyFirst, kbd.DefaultKeyLast); serr != nil {
		return emitError(env.Stdout, internalf("%s", serr.Error()))
	}
	return emitOK(env.Stdout, struct {
		OK bool `json:"ok"`
	}{true})
}
