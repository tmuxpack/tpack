package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

// IdealSize returns the fixed popup dimensions.
func IdealSize(_ *config.Config, _ []plugin.Plugin, _ Deps, _ ...ModelOption) (width, height int) {
	return FixedWidth, FixedHeight
}

// Run launches the TUI with the given configuration and plugins.
func Run(cfg *config.Config, plugins []plugin.Plugin, deps Deps, opts ...ModelOption) error {
	m := NewModel(cfg, plugins, deps, opts...)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
