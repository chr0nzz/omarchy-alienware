package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
)

const introspectXML = `<node>
  <interface name="` + Interface + `">
    <method name="Status">
      <arg direction="out" type="s"/>
    </method>
    <method name="SetProfile">
      <arg direction="in" type="s" name="name"/>
    </method>
    <method name="SetBoost">
      <arg direction="in" type="s" name="fan"/>
      <arg direction="in" type="u" name="value"/>
    </method>
    <method name="ApplyCurve">
      <arg direction="in" type="s" name="json"/>
    </method>
    <method name="StopCurve"/>
    <method name="SetTurbo">
      <arg direction="in" type="b" name="on"/>
    </method>
    <method name="SetPowerLimit">
      <arg direction="in" type="u" name="constraint"/>
      <arg direction="in" type="u" name="watts"/>
    </method>
    <method name="KeyboardStatus">
      <arg direction="out" type="s"/>
    </method>
    <method name="SetKeyboardKeys">
      <arg direction="in" type="s" name="keys"/>
    </method>
    <method name="SetKeyboardAll">
      <arg direction="in" type="s" name="color"/>
    </method>
    <method name="KeyboardOff"/>
  </interface>` + introspect.IntrospectDataString + `</node>`

var ppdNames = []string{
	"net.hadess.PowerProfiles",
	"org.freedesktop.UPower.PowerProfiles",
}

func detectPPD(conn *dbus.Conn) func() bool {
	return func() bool {
		bus := conn.BusObject()
		for _, name := range ppdNames {
			var owned bool
			if err := bus.Call("org.freedesktop.DBus.NameHasOwner", 0, name).Store(&owned); err != nil {
				continue
			}
			if owned {
				return true
			}
		}
		return false
	}
}

func Run(reader *hw.Reader, logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}

	if err := reader.ResetBoost(); err != nil {
		logger.Printf("failsafe: could not reset fan boost at startup: %v", err)
	}

	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("cannot connect to the system bus: %w", err)
	}
	defer conn.Close()

	reader.PPDRunning = detectPPD(conn)
	svc := NewService(reader, &polkitAuthorizer{conn: conn}, logger)

	table := map[string]any{
		"Status":          svc.Status,
		"SetProfile":      svc.SetProfile,
		"SetBoost":        svc.SetBoost,
		"ApplyCurve":      svc.ApplyCurve,
		"StopCurve":       svc.StopCurve,
		"SetTurbo":        svc.SetTurbo,
		"SetPowerLimit":   svc.SetPowerLimit,
		"KeyboardStatus":  svc.KeyboardStatus,
		"SetKeyboardKeys": svc.SetKeyboardKeys,
		"SetKeyboardAll":  svc.SetKeyboardAll,
		"KeyboardOff":     svc.KeyboardOff,
	}
	if err := conn.ExportMethodTable(table, dbus.ObjectPath(ObjectPath), Interface); err != nil {
		return fmt.Errorf("cannot export %s: %w", Interface, err)
	}
	if err := conn.Export(introspect.Introspectable(introspectXML), dbus.ObjectPath(ObjectPath),
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("cannot export introspection data: %w", err)
	}

	reply, err := conn.RequestName(BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("cannot request %s: %w", BusName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("%s is already owned by another process", BusName)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := notify("READY=1"); err != nil {
		logger.Printf("sd_notify READY failed: %v", err)
	}
	logger.Printf("listening on %s at %s", BusName, ObjectPath)

	if interval := watchdogInterval(); interval > 0 {
		go runWatchdog(ctx, interval, logger)
	}

	disconnect := make(chan *dbus.Signal, 1)
	conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/DBus/Local"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Local"),
		dbus.WithMatchMember("Disconnected"),
	)
	conn.Signal(disconnect)

	lost := false
	select {
	case <-ctx.Done():
	case <-disconnect:
		lost = true
		logger.Printf("the system bus connection dropped, stopping so nothing keeps driving the fans")
	}

	if err := notify("STOPPING=1"); err != nil {
		logger.Printf("sd_notify STOPPING failed: %v", err)
	}
	svc.Stop()
	if err := reader.ResetBoost(); err != nil {
		logger.Printf("failsafe: could not reset fan boost on shutdown: %v", err)
	}
	logger.Printf("stopped, fan boost reset to 0")
	if lost {
		return errBusLost
	}
	return nil
}

var errBusLost = errors.New("the system bus connection dropped")

func runWatchdog(ctx context.Context, interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := notify("WATCHDOG=1"); err != nil {
				logger.Printf("sd_notify WATCHDOG failed: %v", err)
			}
		}
	}
}
