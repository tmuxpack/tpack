package tui

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

// Messages returned by operations.
type pluginInstallResultMsg struct {
	Name    string
	Success bool
	Message string
}

type pluginUpdateResultMsg struct {
	Name    string
	Success bool
	Message string
	Output  string
}

type pluginCleanResultMsg struct {
	Name    string
	Success bool
	Message string
}

type pluginUninstallResultMsg struct {
	Name    string
	Success bool
	Message string
}

// installPluginCmd returns a tea.Cmd that clones a plugin.
func installPluginCmd(cloner git.Cloner, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Try raw URL first.
		err := cloner.Clone(ctx, git.CloneOptions{
			URL:    op.Spec,
			Dir:    op.Path,
			Branch: op.Branch,
		})
		if err != nil {
			// Fallback: expand to GitHub URL.
			ghURL := plugin.NormalizeURL(op.Spec)
			err = cloner.Clone(ctx, git.CloneOptions{
				URL:    ghURL,
				Dir:    op.Path,
				Branch: op.Branch,
			})
		}

		if err != nil {
			return pluginInstallResultMsg{
				Name:    op.Name,
				Success: false,
				Message: err.Error(),
			}
		}
		return pluginInstallResultMsg{
			Name:    op.Name,
			Success: true,
			Message: "installed successfully",
		}
	}
}

// updatePluginCmd returns a tea.Cmd that pulls updates for a plugin.
func updatePluginCmd(puller git.Puller, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		output, err := puller.Pull(ctx, git.PullOptions{Dir: op.Path})
		if err != nil {
			return pluginUpdateResultMsg{
				Name:    op.Name,
				Success: false,
				Message: err.Error(),
				Output:  output,
			}
		}
		return pluginUpdateResultMsg{
			Name:    op.Name,
			Success: true,
			Message: "updated successfully",
			Output:  output,
		}
	}
}

// cleanPluginCmd returns a tea.Cmd that removes an orphaned directory.
func cleanPluginCmd(op pendingOp) tea.Cmd {
	return func() tea.Msg {
		err := os.RemoveAll(op.Path)
		if err != nil {
			return pluginCleanResultMsg{
				Name:    op.Name,
				Success: false,
				Message: err.Error(),
			}
		}
		// Verify removal.
		if _, statErr := os.Stat(op.Path); statErr == nil {
			return pluginCleanResultMsg{
				Name:    op.Name,
				Success: false,
				Message: "directory still exists after removal",
			}
		}
		return pluginCleanResultMsg{
			Name:    op.Name,
			Success: true,
			Message: "removed successfully",
		}
	}
}

// uninstallPluginCmd returns a tea.Cmd that removes an installed plugin directory.
func uninstallPluginCmd(op pendingOp) tea.Cmd {
	return func() tea.Msg {
		err := os.RemoveAll(op.Path)
		if err != nil {
			return pluginUninstallResultMsg{
				Name:    op.Name,
				Success: false,
				Message: err.Error(),
			}
		}
		if _, statErr := os.Stat(op.Path); statErr == nil {
			return pluginUninstallResultMsg{
				Name:    op.Name,
				Success: false,
				Message: "directory still exists after removal",
			}
		}
		return pluginUninstallResultMsg{
			Name:    op.Name,
			Success: true,
			Message: "uninstalled successfully",
		}
	}
}

// dispatchNext dispatches the next pending operation, or nil if queue is empty.
func (m *Model) dispatchNext() tea.Cmd {
	if len(m.pendingItems) == 0 {
		m.processing = false
		return nil
	}

	op := m.pendingItems[0]
	m.pendingItems = m.pendingItems[1:]
	m.currentItemName = op.Name

	switch m.operation {
	case OpNone:
		m.processing = false
		return nil
	case OpInstall:
		return installPluginCmd(m.cloner, op)
	case OpUpdate:
		return updatePluginCmd(m.puller, op)
	case OpClean:
		return cleanPluginCmd(op)
	case OpUninstall:
		return uninstallPluginCmd(op)
	}
	m.processing = false
	return nil
}

// buildInstallOps builds the pending operations for install.
func (m *Model) buildInstallOps() []pendingOp {
	indices := m.targetIndices()
	var ops []pendingOp
	for _, i := range indices {
		p := m.plugins[i]
		if p.Status == StatusNotInstalled {
			ops = append(ops, pendingOp{
				Name:   p.Name,
				Spec:   p.Spec,
				Branch: p.Branch,
				Path:   plugin.PluginPath(p.Name, m.cfg.PluginPath),
			})
		}
	}
	return ops
}

// buildUpdateOps builds the pending operations for update.
func (m *Model) buildUpdateOps() []pendingOp {
	indices := m.targetIndices()
	var ops []pendingOp
	for _, i := range indices {
		p := m.plugins[i]
		if p.Status == StatusInstalled {
			ops = append(ops, pendingOp{
				Name:   p.Name,
				Spec:   p.Spec,
				Branch: p.Branch,
				Path:   plugin.PluginPath(p.Name, m.cfg.PluginPath),
			})
		}
	}
	// If nothing selected and no cursor match, update all installed.
	if len(ops) == 0 && !m.multiSelectActive {
		for _, p := range m.plugins {
			if p.Status == StatusInstalled {
				ops = append(ops, pendingOp{
					Name:   p.Name,
					Spec:   p.Spec,
					Branch: p.Branch,
					Path:   plugin.PluginPath(p.Name, m.cfg.PluginPath),
				})
			}
		}
	}
	return ops
}

// buildCleanOps builds the pending operations for clean.
func (m *Model) buildCleanOps() []pendingOp {
	var ops []pendingOp
	for _, o := range m.orphans {
		ops = append(ops, pendingOp{
			Name: o.Name,
			Path: o.Path,
		})
	}
	return ops
}

// buildUninstallOps builds the pending operations for uninstall.
func (m *Model) buildUninstallOps() []pendingOp {
	indices := m.targetIndices()
	var ops []pendingOp
	for _, i := range indices {
		p := m.plugins[i]
		if p.Status == StatusInstalled {
			ops = append(ops, pendingOp{
				Name: p.Name,
				Spec: p.Spec,
				Path: plugin.PluginPath(p.Name, m.cfg.PluginPath),
			})
		}
	}
	return ops
}

// targetIndices returns the indices to operate on: selected if any, else cursor.
func (m *Model) targetIndices() []int {
	if m.multiSelectActive {
		return m.selectedIndices()
	}
	if m.cursor >= 0 && m.cursor < len(m.plugins) {
		return []int{m.cursor}
	}
	return nil
}
