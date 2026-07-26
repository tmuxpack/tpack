package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseSourceDirectives(t *testing.T) {
	content := "source-file -Fqv -t %1 '#{HOME}/one.conf' two.conf\n" +
		"source three.conf four.conf # trailing comment\n" +
		"source-file first.conf \\\n second.conf\n"

	got, err := parseSourceDirectives("/home/user/.tmux.conf", content, func(value string) (string, error) {
		return strings.ReplaceAll(value, "#{HOME}", "/home/user"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{
		{
			Paths:         []string{"#{HOME}/one.conf", "two.conf"},
			Optional:      true,
			ExpandFormats: true,
			Line:          1,
			Text:          "source-file -Fqv -t %1 '#{HOME}/one.conf' two.conf",
		},
		{
			Paths: []string{"three.conf", "four.conf"},
			Line:  2,
			Text:  "source three.conf four.conf",
		},
		{
			Paths: []string{"first.conf", "second.conf"},
			Line:  3,
			Text:  "source-file first.conf \\\n second.conf",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesHandlesCommandSyntax(t *testing.T) {
	content := "# source ignored.conf\n" +
		`source 'semi;colon.conf'; source escaped\;semicolon.conf; ` +
		"source-file -Fnqv -t%1 one.conf; source -qt %2 two.conf # trailing comment\n"

	got, err := parseSourceDirectives("tmux.conf", content, noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{
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
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesTreatsUnquotedHashAsCommentOutsideCondition(t *testing.T) {
	got, err := parseSourceDirectives("tmux.conf", "source valid.conf #{comment}\n", noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{{Paths: []string{"valid.conf"}, Line: 1, Text: "source valid.conf"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
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

func TestParseSourceDirectivesRejectsMalformedSyntax(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		message string
	}{
		{name: "missing target", content: "set -g mouse on\nsource-file -t\n", line: 2, message: "option -t requires a value"},
		{name: "unknown flag", content: "source -x one.conf\n", line: 1, message: "unknown source option -x"},
		{name: "missing path", content: "source-file -Fqv\n", line: 1, message: "requires at least one path"},
		{name: "unmatched single quote", content: "source 'one.conf\n", line: 1, message: "unmatched single quote"},
		{name: "unmatched double quote", content: "source \"one.conf\n", line: 1, message: "unmatched double quote"},
		{name: "dangling continuation", content: "source one.conf \\", line: 1, message: "dangling line continuation"},
		{name: "braced argument", content: "source {one.conf}\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "unsupported directive", content: "%hidden SECRET=value\n", line: 1, message: "unsupported tmux directive %hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSourceDirectives("/tmp/tmux.conf", tt.content, noFormatExpansion)
			assertConfigParseError(t, err, "/tmp/tmux.conf", tt.line, tt.message)
		})
	}
}

func TestParseSourceDirectivesRejectsBracedCommandLists(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "one line", content: "if-shell true { source hidden.conf }\n"},
		{name: "multiline", content: "if-shell true {\nsource hidden.conf\n}\n"},
		{name: "nested", content: "if-shell true {\nif-shell true {\nsource hidden.conf\n}\n}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSourceDirectives("tmux.conf", tt.content, noFormatExpansion)
			assertConfigParseError(t, err, "tmux.conf", 1, "unquoted braced command lists are unsupported")
		})
	}
}

func TestParseSourceDirectivesRejectsNestedExecutableSourceLists(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "then branch",
			content: "if-shell 'test -f ~/.tmux/local.conf' 'source-file ~/.tmux/local.conf'\n",
		},
		{
			name:    "else branch with options",
			content: "if-shell -bF -t %1 '#{enabled}' 'display-message enabled' 'source -q fallback.conf'\n",
		},
		{
			name:    "nested if-shell branch",
			content: `if-shell true "if-shell true 'source-file nested.conf'"` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSourceDirectives("/tmp/tmux.conf", tt.content, noFormatExpansion)
			assertConfigParseError(t, err, "/tmp/tmux.conf", 1, "executable quoted command list may contain source")
		})
	}
}

func TestParseSourceDirectivesAvoidsUnrelatedQuotedSourceStrings(t *testing.T) {
	content := "run-shell 'printf source-file'\n" +
		"set -g @message 'source-file is documented here'\n" +
		"if-shell 'test source-file = source-file' 'display-message safe'\n"

	got, err := parseSourceDirectives("tmux.conf", content, noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("directives = %#v, want none", got)
	}
}

func TestParseSourceDirectivesIgnoresNestedExecutableSourceListInInactiveConditional(t *testing.T) {
	content := "%if 0\nif-shell true 'source-file hidden.conf'\n%endif\n"

	got, err := parseSourceDirectives("tmux.conf", content, noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("directives = %#v, want none", got)
	}
}

func TestParseSourceDirectivesAllowsQuotedBraces(t *testing.T) {
	got, err := parseSourceDirectives("tmux.conf", "source '{literal}.conf'\n", noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{{Paths: []string{"{literal}.conf"}, Line: 1, Text: "source '{literal}.conf'"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesSkipsInactiveConditional(t *testing.T) {
	content := "%if 0\nsource missing.conf\n%else\nsource active.conf\n%endif\n"
	got, err := parseSourceDirectives("tmux.conf", content, noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{{Paths: []string{"active.conf"}, Line: 4, Text: "source active.conf"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesHandlesElifNestingAndInlineConditionals(t *testing.T) {
	content := "%if 0 source ignored.conf %elif enabled source elif.conf %else source else.conf %endif\n" +
		"%if outer\n%if 0\nsource nested-ignored.conf\n%else\nsource nested-active.conf\n%endif\n%endif\n"

	got, err := parseSourceDirectives("tmux.conf", content, noFormatExpansion)
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{
		{Paths: []string{"elif.conf"}, Line: 1, Text: "source elif.conf"},
		{Paths: []string{"nested-active.conf"}, Line: 6, Text: "source nested-active.conf"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesExpandsDynamicConditions(t *testing.T) {
	tests := []struct {
		name     string
		expanded string
		wantPath string
	}{
		{name: "nonzero is true", expanded: "2", wantPath: "active.conf"},
		{name: "zero is false", expanded: "0", wantPath: "fallback.conf"},
		{name: "empty is false", expanded: "", wantPath: "fallback.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "%if #{enabled} source active.conf %else source fallback.conf %endif\n"
			got, err := parseSourceDirectives("tmux.conf", content, func(value string) (string, error) {
				if value != "#{enabled}" {
					return "", fmt.Errorf("unexpected format %q", value)
				}
				return tt.expanded, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0].Paths, []string{tt.wantPath}) {
				t.Fatalf("directives = %#v, want path %q", got, tt.wantPath)
			}
		})
	}
}

func TestParseSourceDirectivesScansBalancedUnquotedConditionalFormats(t *testing.T) {
	const expression = "#{&&:#{==:#{host},example}, #{==:#{pane_id},%1}}"
	tests := []struct {
		name    string
		content string
	}{
		{name: "if", content: "%if " + expression + " source active.conf %endif\n"},
		{name: "elif", content: "%if 0 source ignored.conf %elif " + expression + " source active.conf %endif\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSourceDirectives("tmux.conf", tt.content, func(value string) (string, error) {
				if value != expression {
					return "", fmt.Errorf("format = %q, want %q", value, expression)
				}
				return "1", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0].Paths, []string{"active.conf"}) {
				t.Fatalf("directives = %#v, want active.conf", got)
			}
		})
	}
}

func TestParseSourceDirectivesExpandsConditionsContainingFormatSpans(t *testing.T) {
	tests := []struct {
		name      string
		condition string
	}{
		{name: "multiple spans", condition: "#{first}#{second}"},
		{name: "mixed literal and format", condition: "0#{value}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "%if '" + tt.condition + "' source wrong.conf %else source fallback.conf %endif\n"
			got, err := parseSourceDirectives("tmux.conf", content, func(value string) (string, error) {
				if value != tt.condition {
					return "", fmt.Errorf("format = %q, want %q", value, tt.condition)
				}
				return "0", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0].Paths, []string{"fallback.conf"}) {
				t.Fatalf("directives = %#v, want fallback.conf", got)
			}
		})
	}
}

func TestParseSourceDirectivesRejectsMalformedConditionalFormatSpans(t *testing.T) {
	tests := []struct {
		name      string
		condition string
	}{
		{name: "unclosed span", condition: "#{first"},
		{name: "unclosed second span", condition: "#{first}#{second"},
		{name: "unclosed nested span", condition: "#{outer:#{inner}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "%if '" + tt.condition + "' source wrong.conf %endif\n"
			_, err := parseSourceDirectives("/tmp/tmux.conf", content, noFormatExpansion)
			assertConfigParseError(t, err, "/tmp/tmux.conf", 1, "unbalanced tmux format span")
		})
	}
}

func TestParseSourceDirectivesDoesNotExpandStaticOrInactiveConditions(t *testing.T) {
	content := "%if 1 source static.conf %endif\n" +
		"%if 0 %if #{inactive} source hidden.conf %endif %endif\n"
	got, err := parseSourceDirectives("tmux.conf", content, func(value string) (string, error) {
		t.Fatalf("unexpected expansion of %q", value)
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []sourceDirective{{Paths: []string{"static.conf"}, Line: 1, Text: "source static.conf"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("directives = %#v, want %#v", got, want)
	}
}

func TestParseSourceDirectivesValidatesInactiveConditionalFormatSyntax(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		content := "%if 0\n%if '#{broken'\nsource hidden.conf\n%endif\n%endif\n"
		_, err := parseSourceDirectives("/tmp/tmux.conf", content, func(value string) (string, error) {
			t.Fatalf("unexpected expansion of %q", value)
			return "", nil
		})
		assertConfigParseError(t, err, "/tmp/tmux.conf", 2, "unbalanced tmux format span")
	})

	t.Run("valid is not expanded", func(t *testing.T) {
		content := "%if 0\n%if '0#{value}'\nsource hidden.conf\n%endif\n%endif\n"
		got, err := parseSourceDirectives("tmux.conf", content, func(value string) (string, error) {
			t.Fatalf("unexpected expansion of %q", value)
			return "", nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("directives = %#v, want none", got)
		}
	})
}

func TestParseSourceDirectivesReturnsConditionalErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		message string
	}{
		{name: "expansion failure", content: "%if '#{enabled}'\n%endif\n", line: 1, message: "expand conditional format: unavailable"},
		{name: "unmatched endif", content: "%endif\n", line: 1, message: "unexpected %endif"},
		{name: "unmatched elif", content: "%elif 1\n", line: 1, message: "unexpected %elif"},
		{name: "unmatched else", content: "%else\n", line: 1, message: "unexpected %else"},
		{name: "missing endif", content: "%if 1\nsource one.conf\n", line: 1, message: "missing %endif"},
		{name: "missing if condition", content: "%if\n", line: 1, message: "%if requires a condition"},
		{name: "braced condition", content: "%if {1}\n%endif\n", line: 1, message: "unquoted braced command lists are unsupported"},
		{name: "missing elif condition", content: "%if 0\n%elif\n%endif\n", line: 2, message: "%elif requires a condition"},
		{name: "elif after else", content: "%if 0\n%else\n%elif 1\n%endif\n", line: 3, message: "%elif after %else"},
		{name: "duplicate else", content: "%if 0\n%else\n%else\n%endif\n", line: 3, message: "duplicate %else"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expand := noFormatExpansion
			if tt.name == "expansion failure" {
				expand = func(string) (string, error) { return "", errors.New("unavailable") }
			}
			_, err := parseSourceDirectives("tmux.conf", tt.content, expand)
			assertConfigParseError(t, err, "tmux.conf", tt.line, tt.message)
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
