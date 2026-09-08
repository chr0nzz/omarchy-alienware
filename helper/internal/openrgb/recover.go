package openrgb

import "time"

const resetSettleDelay = 300 * time.Millisecond

type Resetter interface {
	Reset() (string, error)
}

func OpenWithReset(addr string, resetter Resetter) (*Session, error) {
	session, err := Open(addr)
	if err != nil {
		return nil, err
	}
	if len(session.Ctrl.Zones) > 0 {
		return session, nil
	}
	session.Close()
	if resetter == nil {
		return nil, newError(CodeWedged, "the RGB controller reports zero zones and no reset is configured")
	}
	node, err := resetter.Reset()
	if err != nil {
		return nil, newError(CodeWedged, "the RGB controller reports zero zones, and the reset failed: %s", err.Error())
	}
	time.Sleep(resetSettleDelay)
	retried, err := Open(addr)
	if err != nil {
		return nil, err
	}
	if len(retried.Ctrl.Zones) > 0 {
		return retried, nil
	}
	retried.Close()
	return nil, newError(CodeWedged, "the RGB controller at %s still reports zero zones after a reset, it is likely wedged in a way a reset cannot clear", node)
}
