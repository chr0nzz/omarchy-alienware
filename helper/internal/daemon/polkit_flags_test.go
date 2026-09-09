package daemon

import "testing"

func TestInteractionFlagsNeverPromptsForLighting(t *testing.T) {
	cases := []struct {
		action string
		want   uint32
	}{
		{ActionSetKeyboard, flagNoUserInteraction},
		{ActionSetProfile, flagAllowUserInteraction},
		{ActionSetFan, flagAllowUserInteraction},
		{ActionSetPower, flagAllowUserInteraction},
	}
	for _, c := range cases {
		if got := interactionFlags(c.action); got != c.want {
			t.Errorf("interactionFlags(%q) = %d, want %d", c.action, got, c.want)
		}
	}
}
