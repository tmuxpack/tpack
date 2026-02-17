package shell_test

import (
	"testing"

	"github.com/tmuxpack/tpack/internal/shell"
)

func TestQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "hello", "'hello'"},
		{"with single quote", "it's", `'it'\''s'`},
		{"empty", "", "''"},
		{"with null byte", "a\x00b", "'ab'"},
		{"with spaces", "hello world", "'hello world'"},
		{"multiple quotes", "a'b'c", `'a'\''b'\''c'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shell.Quote(tt.in)
			if got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEscapeInSingleQuotes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "hello", "hello"},
		{"with single quote", "it's", `it'\''s`},
		{"empty", "", ""},
		{"with null byte", "a\x00b", "ab"},
		{"with spaces", "hello world", "hello world"},
		{"multiple quotes", "a'b'c", `a'\''b'\''c`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shell.EscapeInSingleQuotes(tt.in)
			if got != tt.want {
				t.Errorf("EscapeInSingleQuotes(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
