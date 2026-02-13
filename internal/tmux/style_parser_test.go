package tmux

import "testing"

func TestParseStyle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  StyleAttrs
	}{
		{
			name:  "fg and bg hex",
			input: "fg=#aabbcc,bg=#112233",
			want:  StyleAttrs{FG: "#aabbcc", BG: "#112233"},
		},
		{
			name:  "bg only",
			input: "bg=red",
			want:  StyleAttrs{BG: "red"},
		},
		{
			name:  "fg only",
			input: "fg=colour123",
			want:  StyleAttrs{FG: "colour123"},
		},
		{
			name:  "with extra attributes",
			input: "fg=#aabbcc,bg=#112233,bold,italics",
			want:  StyleAttrs{FG: "#aabbcc", BG: "#112233"},
		},
		{
			name:  "empty string",
			input: "",
			want:  StyleAttrs{},
		},
		{
			name:  "only attributes no colors",
			input: "bold,italics",
			want:  StyleAttrs{},
		},
		{
			name:  "spaces around parts",
			input: " fg=#abc , bg=#def ",
			want:  StyleAttrs{FG: "#abc", BG: "#def"},
		},
		{
			name:  "default fg and bg",
			input: "fg=default,bg=default",
			want:  StyleAttrs{FG: "default", BG: "default"},
		},
		{
			name:  "named colors",
			input: "fg=white,bg=blue",
			want:  StyleAttrs{FG: "white", BG: "blue"},
		},
		{
			name:  "colour256 values",
			input: "fg=colour45,bg=colour200",
			want:  StyleAttrs{FG: "colour45", BG: "colour200"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStyle(tt.input)
			if got != tt.want {
				t.Errorf("ParseStyle(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "hex color", input: "#aabbcc", want: "#aabbcc"},
		{name: "colour256", input: "colour123", want: "123"},
		{name: "color256 alt spelling", input: "color45", want: "45"},
		{name: "named color", input: "red", want: "red"},
		{name: "default", input: "default", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "whitespace", input: "  #abc  ", want: "#abc"},
		{name: "colour0", input: "colour0", want: "0"},
		{name: "named white", input: "white", want: "white"},
		{name: "hex uppercase", input: "#AABBCC", want: "#AABBCC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeColor(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeColor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
