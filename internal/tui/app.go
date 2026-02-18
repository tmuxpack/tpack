package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
)

// Returns the fixed popup dimensions.
func IdealSize(_ *config.Config, _ []plug.Plugin, _ Deps, _ ...ModelOption) (width, height int) {
	return FixedWidth, FixedHeight
}

// Launches the TUI with the given configuration and plugins.
func Run(cfg *config.Config, plugins []plug.Plugin, deps Deps, opts ...ModelOption) error {
	m := NewModel(cfg, plugins, deps, opts...)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
