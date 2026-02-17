package shell

import "strings"

// Quote wraps s in single quotes for safe use as a POSIX shell argument.
// Null bytes are stripped as they can truncate shell arguments. Single quotes
// are escaped using the '\” break-and-rejoin technique.
func Quote(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// EscapeInSingleQuotes escapes s for safe embedding inside an already
// single-quoted POSIX shell string. Null bytes are stripped and single quotes
// are escaped using the '\” break-and-rejoin technique.
func EscapeInSingleQuotes(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.ReplaceAll(s, "'", "'\\''")
}
