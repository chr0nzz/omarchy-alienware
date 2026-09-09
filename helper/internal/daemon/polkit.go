package daemon

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	polkitService   = "org.freedesktop.PolicyKit1"
	polkitPath      = "/org/freedesktop/PolicyKit1/Authority"
	polkitInterface = "org.freedesktop.PolicyKit1.Authority"

	ActionSetProfile  = "org.xyzlab.alienware.set-profile"
	ActionSetFan      = "org.xyzlab.alienware.set-fan"
	ActionSetPower    = "org.xyzlab.alienware.set-power"
	ActionSetKeyboard = "org.xyzlab.alienware.set-keyboard"

	authorizeTimeout = 30 * time.Second

	flagAllowUserInteraction = uint32(1)
	flagNoUserInteraction    = uint32(0)
)

func interactionFlags(action string) uint32 {
	if action == ActionSetKeyboard {
		return flagNoUserInteraction
	}
	return flagAllowUserInteraction
}

type polkitSubject struct {
	Kind    string
	Details map[string]dbus.Variant
}

type polkitResult struct {
	IsAuthorized bool
	IsChallenge  bool
	Details      map[string]string
}

type authorizer interface {
	Authorize(sender dbus.Sender, action string) error
}

type polkitAuthorizer struct {
	conn *dbus.Conn
}

func (p *polkitAuthorizer) Authorize(sender dbus.Sender, action string) error {
	if sender == "" {
		return errDenied("caller identity is unknown")
	}
	subject := polkitSubject{
		Kind: "system-bus-name",
		Details: map[string]dbus.Variant{
			"name": dbus.MakeVariant(string(sender)),
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), authorizeTimeout)
	defer cancel()
	obj := p.conn.Object(polkitService, dbus.ObjectPath(polkitPath))
	var res polkitResult
	call := obj.CallWithContext(ctx, polkitInterface+".CheckAuthorization", 0,
		subject, action, map[string]string{}, interactionFlags(action), "")
	if call.Err != nil {
		return errDenied("polkit is unavailable: " + call.Err.Error())
	}
	if err := call.Store(&res); err != nil {
		return errDenied("polkit returned an unexpected reply: " + err.Error())
	}
	if !res.IsAuthorized {
		if res.IsChallenge {
			return errDenied("not authorized for " + action + " yet, the session must be unlocked and active")
		}
		return errDenied("not authorized for " + action)
	}
	return nil
}

type allowAll struct{}

func (allowAll) Authorize(dbus.Sender, string) error { return nil }
