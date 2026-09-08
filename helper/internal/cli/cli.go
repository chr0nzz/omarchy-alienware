package cli

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/client"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/daemon"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/elc"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/usbreset"
)

var Version = "0.2.0"

var rgbResetter elc.Resetter = usbreset.New()

var openRGBSessionFunc = func() (*elc.Session, error) {
	return elc.OpenWithReset(rgbResetter)
}

const Usage = `alienwarectl <verb> [args]

  daemon                            run the privileged D-Bus service
  status                            print the full status JSON
  profile <name>                    select a platform profile
  boost <cpu|gpu> <0-255>           set an additive fan boost
  curve apply <file|->              apply a fan curve from JSON
  curve stop                        stop the fan curve and reset boost
  turbo <on|off>                    toggle Intel turbo
  pl <1|2|3> <watts>                set a RAPL power limit
  gpu                               print the gpu object alone
  rgb status                        print the RGB device and its regions
  rgb set <region> <RRGGBB>         light one region: power, logo, ring-top, ring-bottom
  rgb set-all <RRGGBB>              light every region
  rgb set-map <region=RRGGBB,...>   light named regions in one transaction
  rgb mode <name>                   not supported on the alienfx backend yet
  rgb brightness <0-100>            set the overall brightness
  rgb identify <region>             blink one region red
  rgb off                           blank every region
  rgb reset                         USB reset the AW-ELC controller
  kbd status                        print keyboard presence and key count
  kbd set-map <idx=RRGGBB,...>      light named keys in one transaction
  kbd set-all <RRGGBB>              light every key
  kbd off                           blank every key
  reset-fans                        write boost 0 straight to sysfs
  version                           print the binary version
`

type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func Run(args []string) int {
	return RunWith(Env{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin}, args)
}

func RunWith(env Env, args []string) int {
	if len(args) == 0 {
		io.WriteString(env.Stderr, Usage)
		return emitError(env.Stdout, badRequest("a verb is required"))
	}
	switch args[0] {
	case "daemon":
		return runDaemon(env)
	case "reset-fans":
		return runResetFans(env)
	case "version", "--version", "-v":
		return emitOK(env.Stdout, struct {
			OK      bool   `json:"ok"`
			Version string `json:"version"`
		}{true, Version})
	case "help", "--help", "-h":
		io.WriteString(env.Stdout, Usage)
		return 0
	case "rgb":
		return runRGB(env, args[1:])
	}
	return runDaemonVerb(env, args)
}

func runDaemon(env Env) int {
	logger := log.New(env.Stderr, "", log.LstdFlags)
	if err := daemon.Run(hw.New(), logger); err != nil {
		logger.Printf("%v", err)
		return 1
	}
	return 0
}

func runResetFans(env Env) int {
	err := hw.New().ResetBoost()
	if err != nil && errors.Is(err, hw.ErrHardwareMissing) {
		return emitOK(env.Stdout, struct {
			OK    bool `json:"ok"`
			Reset bool `json:"reset"`
		}{true, false})
	}
	if err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK    bool `json:"ok"`
		Reset bool `json:"reset"`
	}{true, true})
}

func runDaemonVerb(env Env, args []string) int {
	c, err := client.Connect()
	if err != nil {
		return emitError(env.Stdout, err)
	}
	switch args[0] {
	case "status":
		payload, err := c.Status()
		if err != nil {
			return emitError(env.Stdout, err)
		}
		return emitRaw(env.Stdout, payload)

	case "gpu":
		payload, err := c.Status()
		if err != nil {
			return emitError(env.Stdout, err)
		}
		var s daemon.Status
		if err := json.Unmarshal([]byte(payload), &s); err != nil {
			return emitError(env.Stdout, internalf("the daemon returned status JSON that could not be parsed: %s", err.Error()))
		}
		return emitOK(env.Stdout, s.GPU)

	case "profile":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("profile takes exactly one name"))
		}
		if err := c.SetProfile(args[1]); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK      bool   `json:"ok"`
			Profile string `json:"profile"`
		}{true, args[1]})

	case "boost":
		if len(args) != 3 {
			return emitError(env.Stdout, badRequest("boost takes a fan and a value, for example boost cpu 120"))
		}
		id := args[1]
		if id != hw.FanCPU && id != hw.FanGPU {
			return emitError(env.Stdout, badRequest("the fan must be cpu or gpu, got %q", id))
		}
		value, err := strconv.Atoi(args[2])
		if err != nil || value < 0 || value > 255 {
			return emitError(env.Stdout, badRequest("the boost must be a whole number 0-255, got %q", args[2]))
		}
		if err := c.SetBoost(id, uint32(value)); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK    bool   `json:"ok"`
			Fan   string `json:"fan"`
			Boost int    `json:"boost"`
		}{true, id, value})

	case "curve":
		return runCurve(env, c, args[1:])

	case "turbo":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("turbo takes on or off"))
		}
		var on bool
		switch strings.ToLower(args[1]) {
		case "on", "1", "true", "enable":
			on = true
		case "off", "0", "false", "disable":
			on = false
		default:
			return emitError(env.Stdout, badRequest("turbo takes on or off, got %q", args[1]))
		}
		if err := c.SetTurbo(on); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK    bool `json:"ok"`
			Turbo bool `json:"turbo"`
		}{true, on})

	case "pl":
		if len(args) != 3 {
			return emitError(env.Stdout, badRequest("pl takes a limit number and a wattage, for example pl 1 45"))
		}
		pl, err := strconv.Atoi(args[1])
		if err != nil || pl < 1 || pl > 3 {
			return emitError(env.Stdout, badRequest("the power limit must be 1, 2 or 3, got %q", args[1]))
		}
		watts, err := strconv.Atoi(args[2])
		if err != nil || watts < 1 {
			return emitError(env.Stdout, badRequest("the wattage must be a positive whole number, got %q", args[2]))
		}
		constraint := pl - 1
		if err := c.SetPowerLimit(uint32(constraint), uint32(watts)); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK         bool `json:"ok"`
			PL         int  `json:"pl"`
			Constraint int  `json:"constraint"`
			Watts      int  `json:"watts"`
		}{true, pl, constraint, watts})

	case "kbd":
		return runKbd(env, c, args[1:])
	}

	io.WriteString(env.Stderr, Usage)
	return emitError(env.Stdout, badRequest("unknown verb %q", args[0]))
}

type curveResult struct {
	Active     bool `json:"active"`
	Interval   int  `json:"interval,omitempty"`
	Hysteresis int  `json:"hysteresis,omitempty"`
}

func runCurve(env Env, c *client.Client, args []string) int {
	if len(args) == 0 {
		return emitError(env.Stdout, badRequest("curve takes apply or stop"))
	}
	switch args[0] {
	case "stop":
		if err := c.StopCurve(); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK    bool        `json:"ok"`
			Curve curveResult `json:"curve"`
		}{true, curveResult{Active: false}})

	case "apply":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("curve apply takes a file path or -"))
		}
		var raw []byte
		var err error
		if args[1] == "-" {
			raw, err = io.ReadAll(env.Stdin)
		} else {
			raw, err = os.ReadFile(args[1])
		}
		if err != nil {
			return emitError(env.Stdout, badRequest("cannot read the curve: %s", err.Error()))
		}
		parsed, err := fan.Parse(raw)
		if err != nil {
			return emitError(env.Stdout, badRequest("%s", err.Error()))
		}
		if err := c.ApplyCurve(string(raw)); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK    bool        `json:"ok"`
			Curve curveResult `json:"curve"`
		}{true, curveResult{Active: true, Interval: parsed.Interval, Hysteresis: parsed.Hysteresis}})
	}
	return emitError(env.Stdout, badRequest("curve takes apply or stop, got %q", args[0]))
}

func runRGB(env Env, args []string) int {
	if len(args) == 0 {
		return emitError(env.Stdout, badRequest("rgb takes status, set, set-all, set-map, mode, brightness, identify, reset or off"))
	}
	switch args[0] {
	case "reset":
		return runRGBReset(env)
	case "mode":
		return emitError(env.Stdout, notSupported("lighting effects are not implemented on the alienfx backend yet"))
	case "status":
		return runRGBStatus(env)
	case "set":
		return runRGBSet(env, args[1:])
	case "set-all":
		return runRGBSetAll(env, args[1:])
	case "set-map":
		return runRGBSetMap(env, args[1:])
	case "brightness":
		return runRGBBrightness(env, args[1:])
	case "identify":
		return runRGBIdentify(env, args[1:])
	case "off":
		return runRGBOff(env)
	}
	return emitError(env.Stdout, badRequest("unknown rgb verb %q", args[0]))
}

func openRGBSession(env Env) (*elc.Session, bool) {
	session, err := openRGBSessionFunc()
	if err != nil {
		emitError(env.Stdout, err)
		return nil, false
	}
	return session, true
}

func runRGBStatus(env Env) int {
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	device, regions := session.Status()
	return emitOK(env.Stdout, struct {
		OK      bool               `json:"ok"`
		Backend string             `json:"backend"`
		Device  elc.DeviceStatus   `json:"device"`
		Regions []elc.RegionStatus `json:"regions"`
	}{true, "alienfx", device, regions})
}

func runRGBSet(env Env, args []string) int {
	if len(args) != 2 {
		return emitError(env.Stdout, badRequest("rgb set takes a region and a colour, for example rgb set logo ff8800"))
	}
	region, rerr := elc.ResolveRegion(args[0])
	if rerr != nil {
		return emitError(env.Stdout, rerr)
	}
	r, g, b, cerr := parseHexColor(args[1])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.SetRegion(region, r, g, b); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK     bool   `json:"ok"`
		Region string `json:"region"`
		Color  string `json:"color"`
	}{true, string(region.ID), hexColor(r, g, b)})
}

func runRGBSetAll(env Env, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("rgb set-all takes a colour, for example rgb set-all ff8800"))
	}
	r, g, b, cerr := parseHexColor(args[0])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.SetAll(r, g, b); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK    bool   `json:"ok"`
		Color string `json:"color"`
	}{true, hexColor(r, g, b)})
}

func parseRegionColorMap(s string) ([]elc.RegionColor, error) {
	parts := strings.Split(s, ",")
	seen := map[elc.RegionID]bool{}
	entries := make([]elc.RegionColor, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			return nil, badRequest("set-map entries must be region=RRGGBB, got %q", part)
		}
		region, rerr := elc.ResolveRegion(strings.TrimSpace(kv[0]))
		if rerr != nil {
			return nil, rerr
		}
		r, g, b, cerr := parseHexColor(kv[1])
		if cerr != nil {
			return nil, cerr
		}
		if seen[region.ID] {
			return nil, badRequest("region %q is repeated in set-map", region.ID)
		}
		seen[region.ID] = true
		entries = append(entries, elc.RegionColor{Region: region, R: r, G: g, B: b})
	}
	if len(entries) == 0 {
		return nil, badRequest("set-map needs at least one region=RRGGBB pair")
	}
	return entries, nil
}

func runRGBSetMap(env Env, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("rgb set-map takes a comma separated list of region=colour pairs, for example rgb set-map logo=ff8800,power=00ff00"))
	}
	entries, perr := parseRegionColorMap(args[0])
	if perr != nil {
		return emitError(env.Stdout, perr)
	}
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.SetMap(entries); err != nil {
		return emitError(env.Stdout, err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[string(e.Region.ID)] = hexColor(e.R, e.G, e.B)
	}
	return emitOK(env.Stdout, struct {
		OK      bool              `json:"ok"`
		Regions map[string]string `json:"regions"`
	}{true, out})
}

func runRGBBrightness(env Env, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("rgb brightness takes a percentage 0-100"))
	}
	percent, perr := strconv.Atoi(args[0])
	if perr != nil || percent < 0 || percent > 100 {
		return emitError(env.Stdout, badRequest("the brightness must be a whole number 0-100, got %q", args[0]))
	}
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.SetBrightness(percent); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK         bool `json:"ok"`
		Brightness int  `json:"brightness"`
	}{true, percent})
}

func runRGBIdentify(env Env, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("rgb identify takes a region"))
	}
	region, rerr := elc.ResolveRegion(args[0])
	if rerr != nil {
		return emitError(env.Stdout, rerr)
	}
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.Identify(region, 3, 250*time.Millisecond); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK     bool   `json:"ok"`
		Region string `json:"region"`
	}{true, string(region.ID)})
}

func runRGBOff(env Env) int {
	session, ok := openRGBSession(env)
	if !ok {
		return 1
	}
	defer session.Close()
	if err := session.Off(); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK bool `json:"ok"`
	}{true})
}

func runRGBReset(env Env) int {
	node, err := rgbResetter.Reset()
	if err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK   bool   `json:"ok"`
		Node string `json:"node"`
	}{true, node})
}

func runKbd(env Env, c *client.Client, args []string) int {
	if len(args) == 0 {
		return emitError(env.Stdout, badRequest("kbd takes status, set-map, set-all or off"))
	}
	switch args[0] {
	case "status":
		return runKbdStatus(env, c)
	case "set-map":
		return runKbdSetMap(env, c, args[1:])
	case "set-all":
		return runKbdSetAll(env, c, args[1:])
	case "off":
		return runKbdOff(env, c)
	}
	return emitError(env.Stdout, badRequest("unknown kbd verb %q", args[0]))
}

func runKbdStatus(env Env, c *client.Client) int {
	payload, err := c.KeyboardStatus()
	if err != nil {
		return emitError(env.Stdout, err)
	}
	return emitRaw(env.Stdout, payload)
}

func parseKeyColorMapDisplay(s string) (map[string]string, error) {
	parts := strings.Split(s, ",")
	out := map[string]string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			return nil, badRequest("kbd set-map entries must be idx=RRGGBB, got %q", part)
		}
		idx := strings.TrimSpace(kv[0])
		if _, err := strconv.Atoi(idx); err != nil {
			return nil, badRequest("kbd set-map key index must be a whole number, got %q", idx)
		}
		r, g, b, cerr := parseHexColor(kv[1])
		if cerr != nil {
			return nil, cerr
		}
		out[idx] = hexColor(r, g, b)
	}
	if len(out) == 0 {
		return nil, badRequest("kbd set-map needs at least one idx=RRGGBB pair")
	}
	return out, nil
}

func runKbdSetMap(env Env, c *client.Client, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("kbd set-map takes a comma separated list of idx=RRGGBB pairs, for example kbd set-map 0=ff0000,4=00ff00"))
	}
	keys, perr := parseKeyColorMapDisplay(args[0])
	if perr != nil {
		return emitError(env.Stdout, perr)
	}
	if err := c.SetKeyboardKeys(args[0]); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK   bool              `json:"ok"`
		Keys map[string]string `json:"keys"`
	}{true, keys})
}

func runKbdSetAll(env Env, c *client.Client, args []string) int {
	if len(args) != 1 {
		return emitError(env.Stdout, badRequest("kbd set-all takes a colour, for example kbd set-all ff0000"))
	}
	r, g, b, cerr := parseHexColor(args[0])
	if cerr != nil {
		return emitError(env.Stdout, cerr)
	}
	if err := c.SetKeyboardAll(args[0]); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK    bool   `json:"ok"`
		Color string `json:"color"`
	}{true, hexColor(r, g, b)})
}

func runKbdOff(env Env, c *client.Client) int {
	if err := c.KeyboardOff(); err != nil {
		return emitError(env.Stdout, err)
	}
	return emitOK(env.Stdout, struct {
		OK bool `json:"ok"`
	}{true})
}
