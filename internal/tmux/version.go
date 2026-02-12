package tmux

import "unicode"

// ParseVersionDigits extracts digits from a tmux version string
// and returns them as an integer. For example:
//
//	"tmux 1.9"  → 19
//	"tmux 3.4"  → 34
//	"1.9a"      → 19
//	"tmux 3.3a" → 33
func ParseVersionDigits(s string) int {
	result := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			result = result*10 + int(r-'0')
		}
	}
	return result
}

// IsVersionSupported returns true if current >= minimum.
func IsVersionSupported(current, minimum int) bool {
	return current >= minimum
}
