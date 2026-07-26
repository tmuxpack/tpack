package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseSourceDirectives(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		expand  func(string) (string, error)
		want    []sourceDirective
	}{
		{
			name: "source options paths and continuations",
			path: "/home/user/.tmux.conf",
			content: "source-file -Fqv -t %1 '#{HOME}/one.conf' two.conf\n" +
				"source three.conf four.conf # trailing comment\n" +
				"source-file first.conf \\\n second.conf\n",
			expand: func(value string) (string, error) {
				return strings.ReplaceAll(value, "#{HOME}", "/home/user"), nil
			},
			want: []sourceDirective{
				{
					Paths:         []string{"#{HOME}/one.conf", "two.conf"},
					Optional:      true,
					ExpandFormats: true,
					Line:          1,
					Text:          "source-file -Fqv -t %1 '#{HOME}/one.conf' two.conf",
				},
				{Paths: []string{"three.conf", "four.conf"}, Line: 2, Text: "source three.conf four.conf"},
				{Paths: []string{"first.conf", "second.conf"}, Line: 3, Text: "source-file first.conf \\\n second.conf"},
			},
		},
		{
			name: "command syntax",
			path: "tmux.conf",
			content: "# source ignored.conf\n" +
				`source 'semi;colon.conf'; source escaped\;semicolon.conf; ` +
				"source-file -Fnqv -t%1 one.conf; source -qt %2 two.conf # trailing comment\n",
			expand: noFormatExpansion,
			want: []sourceDirective{
				{Paths: []string{"semi;colon.conf"}, Line: 2, Text: "source 'semi;colon.conf'"},
				{Paths: []string{"escaped;semicolon.conf"}, Line: 2, Text: `source escaped\;semicolon.conf`},
				{
					Paths:         []string{"one.conf"},
					Optional:      true,
					ExpandFormats: true,
					ParseOnly:     true,
					Line:          2,
					Text:          "source-file -Fnqv -t%1 one.conf",
				},
				{Paths: []string{"two.conf"}, Optional: true, Line: 2, Text: "source -qt %2 two.conf"},
			},
		},
		{
			name:    "unquoted hash starts comment",
			path:    "tmux.conf",
			content: "source valid.conf #{comment}\n",
			expand:  noFormatExpansion,
			want:    []sourceDirective{{Paths: []string{"valid.conf"}, Line: 1, Text: "source valid.conf"}},
		},
		{
			name: "unrelated quoted source text",
			path: "tmux.conf",
			content: "run-shell 'printf source-file'\n" +
				"set -g @message 'source-file is documented here'\n" +
				"if-shell 'test source-file = source-file' 'display-message safe'\n",
			expand: noFormatExpansion,
			want:   []sourceDirective{},
		},
		{
			name:    "inactive executable list",
			path:    "tmux.conf",
			content: "%if 0\nif-shell true 'source-file hidden.conf'\n%endif\n",
			expand:  noFormatExpansion,
			want:    []sourceDirective{},
		},
		{
			name:    "quoted braces",
			path:    "tmux.conf",
			content: "source '{literal}.conf'\n",
			expand:  noFormatExpansion,
			want:    []sourceDirective{{Paths: []string{"{literal}.conf"}, Line: 1, Text: "source '{literal}.conf'"}},
		},
		{
			name:    "inactive branch",
			path:    "tmux.conf",
			content: "%if 0\nsource missing.conf\n%else\nsource active.conf\n%endif\n",
			expand:  noFormatExpansion,
			want:    []sourceDirective{{Paths: []string{"active.conf"}, Line: 4, Text: "source active.conf"}},
		},
		{
			name: "elif nesting and inline conditionals",
			path: "tmux.conf",
			content: "%if 0 source ignored.conf %elif enabled source elif.conf %else source else.conf %endif\n" +
				"%if outer\n%if 0\nsource nested-ignored.conf\n%else\nsource nested-active.conf\n%endif\n%endif\n",
			expand: noFormatExpansion,
			want: []sourceDirective{
				{Paths: []string{"elif.conf"}, Line: 1, Text: "source elif.conf"},
				{Paths: []string{"nested-active.conf"}, Line: 6, Text: "source nested-active.conf"},
			},
		},
		{
			name: "static and inactive conditions are not expanded",
			path: "tmux.conf",
			content: "%if 1 source static.conf %endif\n" +
				"%if 0 %if #{inactive} source hidden.conf %endif %endif\n",
			expand: func(value string) (string, error) {
				t.Fatalf("unexpected expansion of %q", value)
				return "", nil
			},
			want: []sourceDirective{{Paths: []string{"static.conf"}, Line: 1, Text: "source static.conf"}},
		},
		{
			name:    "valid inactive format is not expanded",
			path:    "tmux.conf",
			content: "%if 0\n%if '0#{value}'\nsource hidden.conf\n%endif\n%endif\n",
			expand: func(value string) (string, error) {
				t.Fatalf("unexpected expansion of %q", value)
				return "", nil
			},
			want: []sourceDirective{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSourceDirectives(tt.path, tt.content, tt.expand)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("directives = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSplitTmuxCommandLinePreservesWriterHashSemantics(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{name: "hash inside word", line: "run foo#{bar} # trailing", want: []string{"run", "foo#{bar}"}},
		{name: "hash starts comment", line: "run foo #{comment}", want: []string{"run", "foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := splitTmuxCommandLine(tt.line)
			if !ok || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitTmuxCommandLine(%q) = (%v, %v), want (%v, true)", tt.line, got, ok, tt.want)
			}
		})
	}
}

func TestScanTmuxCommandsBracePolicy(t *testing.T) {
	t.Run("top-level unquoted braced command list is rejected", func(t *testing.T) {
		_, err := scanTmuxCommands("tmux.conf", "if-shell true { source hidden.conf }\n", true)
		assertConfigParseError(t, err, "tmux.conf", 1, "unquoted braced command lists are unsupported")
	})

	t.Run("quoted nested command list permits braces", func(t *testing.T) {
		commands, err := scanTmuxCommands("", "if-shell true { source hidden.conf }", false)
		if err != nil {
			t.Fatal(err)
		}
		if len(commands) != 1 {
			t.Fatalf("commands = %#v, want one command", commands)
		}
		var got []string
		for _, token := range commands[0].tokens {
			got = append(got, token.value)
		}
		want := []string{"if-shell", "true", "{", "source", "hidden.conf", "}"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("token values = %#v, want %#v", got, want)
		}
	})
}

func TestParseSourceDirectivesExpandsDynamicConditions(t *testing.T) {
	const balancedFormat = "#{&&:#{==:#{host},example}, #{==:#{pane_id},%1}}"
	tests := []struct {
		name           string
		content        string
		expectedFormat string
		expanded       string
		want           []sourceDirective
	}{
		{name: "nonzero is true", content: "%if #{enabled} source active.conf %else source fallback.conf %endif\n", expectedFormat: "#{enabled}", expanded: "2", want: []sourceDirective{{Paths: []string{"active.conf"}, Line: 1, Text: "source active.conf"}}},
		{name: "zero is false", content: "%if #{enabled} source active.conf %else source fallback.conf %endif\n", expectedFormat: "#{enabled}", expanded: "0", want: []sourceDirective{{Paths: []string{"fallback.conf"}, Line: 1, Text: "source fallback.conf"}}},
		{name: "empty is false", content: "%if #{enabled} source active.conf %else source fallback.conf %endif\n", expectedFormat: "#{enabled}", expanded: "", want: []sourceDirective{{Paths: []string{"fallback.conf"}, Line: 1, Text: "source fallback.conf"}}},
		{name: "balanced if format", content: "%if " + balancedFormat + " source active.conf %endif\n", expectedFormat: balancedFormat, expanded: "1", want: []sourceDirective{{Paths: []string{"active.conf"}, Line: 1, Text: "source active.conf"}}},
		{name: "balanced elif format", content: "%if 0 source ignored.conf %elif " + balancedFormat + " source active.conf %endif\n", expectedFormat: balancedFormat, expanded: "1", want: []sourceDirective{{Paths: []string{"active.conf"}, Line: 1, Text: "source active.conf"}}},
		{name: "multiple format spans", content: "%if '#{first}#{second}' source wrong.conf %else source fallback.conf %endif\n", expectedFormat: "#{first}#{second}", expanded: "0", want: []sourceDirective{{Paths: []string{"fallback.conf"}, Line: 1, Text: "source fallback.conf"}}},
		{name: "mixed literal and format", content: "%if '0#{value}' source wrong.conf %else source fallback.conf %endif\n", expectedFormat: "0#{value}", expanded: "0", want: []sourceDirective{{Paths: []string{"fallback.conf"}, Line: 1, Text: "source fallback.conf"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSourceDirectives("tmux.conf", tt.content, func(value string) (string, error) {
				if value != tt.expectedFormat {
					return "", fmt.Errorf("format = %q, want %q", value, tt.expectedFormat)
				}
				return tt.expanded, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("directives = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseSourceDirectivesReturnsErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		expand  func(string) (string, error)
		line    int
		message string
	}{
		{name: "missing target", path: "/tmp/tmux.conf", content: "set -g mouse on\nsource-file -t\n", line: 2, message: "option -t requires a value"},
		{name: "unknown flag", path: "/tmp/tmux.conf", content: "source -x one.conf\n", line: 1, message: "unknown source option -x"},
		{name: "missing path", path: "/tmp/tmux.conf", content: "source-file -Fqv\n", line: 1, message: "requires at least one path"},
		{name: "unmatched single quote", path: "/tmp/tmux.conf", content: "source 'one.conf\n", line: 1, message: "unmatched single quote"},
		{name: "unmatched double quote", path: "/tmp/tmux.conf", content: "source \"one.conf\n", line: 1, message: "unmatched double quote"},
		{name: "dangling continuation", path: "/tmp/tmux.conf", content: "source one.conf \\", line: 1, message: "dangling line continuation"},
		{name: "braced argument", path: "/tmp/tmux.conf", content: "source {one.conf}\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "unsupported directive", path: "/tmp/tmux.conf", content: "%hidden SECRET=value\n", line: 1, message: "unsupported tmux directive %hidden"},
		{name: "one-line braced command list", path: "tmux.conf", content: "if-shell true { source hidden.conf }\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "multiline braced command list", path: "tmux.conf", content: "if-shell true {\nsource hidden.conf\n}\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "nested braced command list", path: "tmux.conf", content: "if-shell true {\nif-shell true {\nsource hidden.conf\n}\n}\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "source in then branch", path: "/tmp/tmux.conf", content: "if-shell 'test -f ~/.tmux/local.conf' 'source-file ~/.tmux/local.conf'\n", line: 1, message: "executable quoted command list may contain source"},
		{name: "source in else branch with options", path: "/tmp/tmux.conf", content: "if-shell -bF -t %1 '#{enabled}' 'display-message enabled' 'source -q fallback.conf'\n", line: 1, message: "executable quoted command list may contain source"},
		{name: "source in nested if-shell branch", path: "/tmp/tmux.conf", content: `if-shell true "if-shell true 'source-file nested.conf'"` + "\n", line: 1, message: "executable quoted command list may contain source"},
		{name: "unclosed format span", path: "/tmp/tmux.conf", content: "%if '#{first' source wrong.conf %endif\n", line: 1, message: "unbalanced tmux format span"},
		{name: "unclosed second format span", path: "/tmp/tmux.conf", content: "%if '#{first}#{second' source wrong.conf %endif\n", line: 1, message: "unbalanced tmux format span"},
		{name: "unclosed nested format span", path: "/tmp/tmux.conf", content: "%if '#{outer:#{inner}' source wrong.conf %endif\n", line: 1, message: "unbalanced tmux format span"},
		{
			name:    "malformed inactive format",
			path:    "/tmp/tmux.conf",
			content: "%if 0\n%if '#{broken'\nsource hidden.conf\n%endif\n%endif\n",
			expand: func(value string) (string, error) {
				t.Fatalf("unexpected expansion of %q", value)
				return "", nil
			},
			line:    2,
			message: "unbalanced tmux format span",
		},
		{name: "expansion failure", path: "tmux.conf", content: "%if '#{enabled}'\n%endif\n", expand: func(string) (string, error) { return "", errors.New("unavailable") }, line: 1, message: "expand conditional format: unavailable"},
		{name: "unmatched endif", path: "tmux.conf", content: "%endif\n", line: 1, message: "unexpected %endif"},
		{name: "unmatched elif", path: "tmux.conf", content: "%elif 1\n", line: 1, message: "unexpected %elif"},
		{name: "unmatched else", path: "tmux.conf", content: "%else\n", line: 1, message: "unexpected %else"},
		{name: "missing endif", path: "tmux.conf", content: "%if 1\nsource one.conf\n", line: 1, message: "missing %endif"},
		{name: "missing if condition", path: "tmux.conf", content: "%if\n", line: 1, message: "%if requires a condition"},
		{name: "braced condition", path: "tmux.conf", content: "%if {1}\n%endif\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "missing elif condition", path: "tmux.conf", content: "%if 0\n%elif\n%endif\n", line: 2, message: "%elif requires a condition"},
		{name: "elif after else", path: "tmux.conf", content: "%if 0\n%else\n%elif 1\n%endif\n", line: 3, message: "%elif after %else"},
		{name: "duplicate else", path: "tmux.conf", content: "%if 0\n%else\n%else\n%endif\n", line: 3, message: "duplicate %else"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expand := tt.expand
			if expand == nil {
				expand = noFormatExpansion
			}
			_, err := parseSourceDirectives(tt.path, tt.content, expand)
			assertConfigParseError(t, err, tt.path, tt.line, tt.message)
		})
	}
}

func noFormatExpansion(value string) (string, error) { return value, nil }

func assertConfigParseError(t *testing.T, err error, path string, line int, message string) {
	t.Helper()
	var parseErr *ConfigParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v, want ConfigParseError", err)
	}
	if parseErr.Path != path || parseErr.Line != line || !strings.Contains(parseErr.Msg, message) {
		t.Fatalf("error = %#v, want path %q, line %d, message containing %q", parseErr, path, line, message)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%s:%d", path, line)) {
		t.Fatalf("error text = %q, want file and line", err)
	}
}
