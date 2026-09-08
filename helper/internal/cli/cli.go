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
	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/openrgb"
)

var Version = "0.1.0"

const Usage = `alienwarectl <verb> [args]

  daemon                         run the privileged D-Bus service
  status                         print the full status JSON
  profile <name>                 select a platform profile
  boost <cpu|gpu> <0-255>        set an additive fan boost
  curve apply <file|->           apply a fan curve from JSON
  curve stop                     stop the fan curve and reset boost
  turbo <on|off>                 toggle Intel turbo
  pl <1|2> <watts>               set a RAPL power limit
  gpu                            print the gpu object alone
  rgb status                     print the RGB device and zones
  rgb set <zone> <RRGGBB>        light one zone
  rgb set-all <RRGGBB>           light every zone
  rgb mode <name>                select a lighting mode
  rgb brightness <0-100>         set the active mode brightness
  rgb identify <zone>            blink one zone red
  rgb off                        blank every LED
  reset-fans                     write boost 0 straight to sysfs
  version                        print the binary version
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
		if err != nil || (pl != 1 && pl != 2) {
			return emitError(env.Stdout, badRequest("the power limit must be 1 or 2, got %q", args[1]))
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
		return emitError(env.Stdout, badRequest("rgb takes status, set, set-all, mode, brightness, identify or off"))
	}
	session, err := openrgb.Open(openrgb.DefaultAddr)
	if err != nil {
		return emitError(env.Stdout, err)
	}
	defer session.Close()

	switch args[0] {
	case "status":
		return emitOK(env.Stdout, session.Status())

	case "set":
		if len(args) != 3 {
			return emitError(env.Stdout, badRequest("rgb set takes a zone and a colour, for example rgb set 0 ff8800"))
		}
		index, err := openrgb.ResolveZone(session.Ctrl, args[1])
		if err != nil {
			return emitError(env.Stdout, err)
		}
		color, err := openrgb.ParseHex(args[2])
		if err != nil {
			return emitError(env.Stdout, badRequest("%s", err.Error()))
		}
		if err := session.SetZone(index, color); err != nil {
			return emitError(env.Stdout, err)
		}
		zone, _ := session.ZoneSummary(index)
		return emitOK(env.Stdout, struct {
			OK    bool               `json:"ok"`
			Zone  openrgb.StatusZone `json:"zone"`
			Color string             `json:"color"`
		}{true, zone, openrgb.HexString(color)})

	case "set-all":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("rgb set-all takes a colour, for example rgb set-all ff8800"))
		}
		color, err := openrgb.ParseHex(args[1])
		if err != nil {
			return emitError(env.Stdout, badRequest("%s", err.Error()))
		}
		if err := session.SetAll(color); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK    bool   `json:"ok"`
			Color string `json:"color"`
		}{true, openrgb.HexString(color)})

	case "mode":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("rgb mode takes a mode name"))
		}
		name, err := session.SetMode(args[1])
		if err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK   bool   `json:"ok"`
			Mode string `json:"mode"`
		}{true, name})

	case "brightness":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("rgb brightness takes a percentage 0-100"))
		}
		percent, err := strconv.Atoi(args[1])
		if err != nil || percent < 0 || percent > 100 {
			return emitError(env.Stdout, badRequest("the brightness must be a whole number 0-100, got %q", args[1]))
		}
		value, err := session.SetBrightness(percent)
		if err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK         bool `json:"ok"`
			Brightness int  `json:"brightness"`
			Value      int  `json:"value"`
		}{true, percent, int(value)})

	case "identify":
		if len(args) != 2 {
			return emitError(env.Stdout, badRequest("rgb identify takes a zone"))
		}
		index, err := openrgb.ResolveZone(session.Ctrl, args[1])
		if err != nil {
			return emitError(env.Stdout, err)
		}
		if err := session.Identify(index, 3, 250*time.Millisecond); err != nil {
			return emitError(env.Stdout, err)
		}
		zone, _ := session.ZoneSummary(index)
		return emitOK(env.Stdout, struct {
			OK   bool               `json:"ok"`
			Zone openrgb.StatusZone `json:"zone"`
		}{true, zone})

	case "off":
		if err := session.Off(); err != nil {
			return emitError(env.Stdout, err)
		}
		return emitOK(env.Stdout, struct {
			OK bool `json:"ok"`
		}{true})
	}
	return emitError(env.Stdout, badRequest("unknown rgb verb %q", args[0]))
}
