package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

func TestBuildTheme_NilRunner(t *testing.T) {
	th := BuildTheme(nil)
	def := DefaultTheme()

	if th.PrimaryColor != def.PrimaryColor {
		t.Errorf("PrimaryColor = %q, want %q", th.PrimaryColor, def.PrimaryColor)
	}
	if th.SecondaryColor != def.SecondaryColor {
		t.Errorf("SecondaryColor = %q, want %q", th.SecondaryColor, def.SecondaryColor)
	}
	if th.AccentColor != def.AccentColor {
		t.Errorf("AccentColor = %q, want %q", th.AccentColor, def.AccentColor)
	}
	if th.ErrorColor != def.ErrorColor {
		t.Errorf("ErrorColor = %q, want %q", th.ErrorColor, def.ErrorColor)
	}
	if th.MutedColor != def.MutedColor {
		t.Errorf("MutedColor = %q, want %q", th.MutedColor, def.MutedColor)
	}
	if th.TextColor != def.TextColor {
		t.Errorf("TextColor = %q, want %q", th.TextColor, def.TextColor)
	}
}

func TestBuildTheme_FullTheme(t *testing.T) {
	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#aaaaaa,bg=#bbbbbb"
	m.Options["pane-active-border-style"] = "fg=#cccccc"
	m.Options["window-status-current-style"] = "bg=#dddddd"

	th := BuildTheme(m)

	if th.PrimaryColor != lipgloss.Color("#bbbbbb") {
		t.Errorf("PrimaryColor = %q, want %q", th.PrimaryColor, "#bbbbbb")
	}
	if th.TextColor != lipgloss.Color("#aaaaaa") {
		t.Errorf("TextColor = %q, want %q", th.TextColor, "#aaaaaa")
	}
	if th.SecondaryColor != lipgloss.Color("#cccccc") {
		t.Errorf("SecondaryColor = %q, want %q", th.SecondaryColor, "#cccccc")
	}
	if th.AccentColor != lipgloss.Color("#dddddd") {
		t.Errorf("AccentColor = %q, want %q", th.AccentColor, "#dddddd")
	}
}

func TestBuildTheme_PartialTheme(t *testing.T) {
	def := DefaultTheme()

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#111111,bg=#222222"

	th := BuildTheme(m)

	if th.PrimaryColor != lipgloss.Color("#222222") {
		t.Errorf("PrimaryColor = %q, want %q", th.PrimaryColor, "#222222")
	}
	if th.TextColor != lipgloss.Color("#111111") {
		t.Errorf("TextColor = %q, want %q", th.TextColor, "#111111")
	}
	if th.SecondaryColor != def.SecondaryColor {
		t.Errorf("SecondaryColor = %q, want default %q", th.SecondaryColor, def.SecondaryColor)
	}
	if th.AccentColor != def.AccentColor {
		t.Errorf("AccentColor = %q, want default %q", th.AccentColor, def.AccentColor)
	}
}

func TestBuildTheme_ErrorColor_Unchanged(t *testing.T) {
	def := DefaultTheme()

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=#ffffff,bg=#000000"

	th := BuildTheme(m)

	if th.ErrorColor != def.ErrorColor {
		t.Errorf("ErrorColor = %q, want default %q", th.ErrorColor, def.ErrorColor)
	}
	if th.MutedColor != def.MutedColor {
		t.Errorf("MutedColor = %q, want default %q", th.MutedColor, def.MutedColor)
	}
}

func TestBuildTheme_DefaultTmuxColors(t *testing.T) {
	def := DefaultTheme()

	m := tmux.NewMockRunner()
	m.Options["status-style"] = "fg=default,bg=default"

	th := BuildTheme(m)

	if th.PrimaryColor != def.PrimaryColor {
		t.Errorf("PrimaryColor = %q, want default %q", th.PrimaryColor, def.PrimaryColor)
	}
	if th.TextColor != def.TextColor {
		t.Errorf("TextColor = %q, want default %q", th.TextColor, def.TextColor)
	}
}

func TestOverlayConfigColors_FullOverride(t *testing.T) {
	base := DefaultTheme()

	th := OverlayConfigColors(base, config.ColorConfig{
		Primary:   "#aa0000",
		Secondary: "#00bb00",
		Accent:    "#0000cc",
		Error:     "#dd0000",
		Muted:     "#555555",
		Text:      "#ffffff",
	})

	if th.PrimaryColor != lipgloss.Color("#aa0000") {
		t.Errorf("PrimaryColor = %q, want %q", th.PrimaryColor, "#aa0000")
	}
	if th.SecondaryColor != lipgloss.Color("#00bb00") {
		t.Errorf("SecondaryColor = %q, want %q", th.SecondaryColor, "#00bb00")
	}
	if th.AccentColor != lipgloss.Color("#0000cc") {
		t.Errorf("AccentColor = %q, want %q", th.AccentColor, "#0000cc")
	}
	if th.ErrorColor != lipgloss.Color("#dd0000") {
		t.Errorf("ErrorColor = %q, want %q", th.ErrorColor, "#dd0000")
	}
	if th.MutedColor != lipgloss.Color("#555555") {
		t.Errorf("MutedColor = %q, want %q", th.MutedColor, "#555555")
	}
	if th.TextColor != lipgloss.Color("#ffffff") {
		t.Errorf("TextColor = %q, want %q", th.TextColor, "#ffffff")
	}
}

func TestOverlayConfigColors_PartialOverride(t *testing.T) {
	base := DefaultTheme()

	th := OverlayConfigColors(base, config.ColorConfig{
		Primary: "#abcdef",
	})

	if th.PrimaryColor != lipgloss.Color("#abcdef") {
		t.Errorf("PrimaryColor = %q, want %q", th.PrimaryColor, "#abcdef")
	}
	if th.SecondaryColor != base.SecondaryColor {
		t.Errorf("SecondaryColor = %q, want base %q", th.SecondaryColor, base.SecondaryColor)
	}
	if th.AccentColor != base.AccentColor {
		t.Errorf("AccentColor = %q, want base %q", th.AccentColor, base.AccentColor)
	}
	if th.ErrorColor != base.ErrorColor {
		t.Errorf("ErrorColor = %q, want base %q", th.ErrorColor, base.ErrorColor)
	}
	if th.MutedColor != base.MutedColor {
		t.Errorf("MutedColor = %q, want base %q", th.MutedColor, base.MutedColor)
	}
	if th.TextColor != base.TextColor {
		t.Errorf("TextColor = %q, want base %q", th.TextColor, base.TextColor)
	}
}

func TestOverlayConfigColors_EmptyNoOp(t *testing.T) {
	base := DefaultTheme()

	th := OverlayConfigColors(base, config.ColorConfig{})

	if th.PrimaryColor != base.PrimaryColor {
		t.Errorf("PrimaryColor = %q, want base %q", th.PrimaryColor, base.PrimaryColor)
	}
	if th.SecondaryColor != base.SecondaryColor {
		t.Errorf("SecondaryColor = %q, want base %q", th.SecondaryColor, base.SecondaryColor)
	}
	if th.AccentColor != base.AccentColor {
		t.Errorf("AccentColor = %q, want base %q", th.AccentColor, base.AccentColor)
	}
	if th.ErrorColor != base.ErrorColor {
		t.Errorf("ErrorColor = %q, want base %q", th.ErrorColor, base.ErrorColor)
	}
	if th.MutedColor != base.MutedColor {
		t.Errorf("MutedColor = %q, want base %q", th.MutedColor, base.MutedColor)
	}
	if th.TextColor != base.TextColor {
		t.Errorf("TextColor = %q, want base %q", th.TextColor, base.TextColor)
	}
}
