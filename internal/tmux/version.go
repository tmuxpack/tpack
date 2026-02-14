package tmux

import (
	"regexp"
	"strconv"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

// ParseVersionDigits extracts a semantic version from a tmux version
// string and encodes it as major*100 + minor. For example:
//
//	"tmux 1.9"   → 109
//	"tmux 3.4"   → 304
//	"tmux 3.10"  → 310
//	"1.9a"       → 109
//	"tmux 3.3a"  → 303
func ParseVersionDigits(s string) int {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major*100 + minor
}

// IsVersionSupported returns true if current >= minimum.
func IsVersionSupported(current, minimum int) bool {
	return current >= minimum
}
