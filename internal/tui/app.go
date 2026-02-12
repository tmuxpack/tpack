package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

// IdealSize computes the ideal popup dimensions by rendering the actual view.
func IdealSize(
	cfg *config.Config,
	plugins []plugin.Plugin,
	cloner git.Cloner,
	puller git.Puller,
	validator git.Validator,
) (width, height int) {
	m := NewModel(cfg, plugins, cloner, puller, validator)
	m.width = 80
	m.viewHeight = len(m.plugins) + 10

	rendered := m.View()
	lines := strings.Split(rendered, "\n")
	height = len(lines)

	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > width {
			width = w
		}
	}

	// Small margin so the popup border doesn't clip content.
	width += 4
	height += 2

	return width, height
}

// Run launches the TUI with the given configuration and plugins.
func Run(
	cfg *config.Config,
	plugins []plugin.Plugin,
	cloner git.Cloner,
	puller git.Puller,
	validator git.Validator,
) error {
	m := NewModel(cfg, plugins, cloner, puller, validator)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
