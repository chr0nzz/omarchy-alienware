package cli

import (
	"fmt"
	"strconv"
	"strings"
)

func parseHexColor(s string) (r, g, b uint8, err error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, badRequest("colour %q must be 6 hex digits, for example ff8800", s)
	}
	v, parseErr := strconv.ParseUint(s, 16, 32)
	if parseErr != nil {
		return 0, 0, 0, badRequest("colour %q must be 6 hex digits, for example ff8800", s)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}

func hexColor(r, g, b uint8) string {
	return fmt.Sprintf("%02x%02x%02x", r, g, b)
}
