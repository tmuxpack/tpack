package tui

import (
	"context"
	"os"

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

type pluginCheckResultMsg struct {
	Name     string
	Outdated bool
	Err      error
}

// checkPluginCmd returns a tea.Cmd that checks if a plugin is outdated.
func checkPluginCmd(fetcher git.Fetcher, name string, dir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CheckTimeout)
		defer cancel()

		outdated, err := fetcher.IsOutdated(ctx, dir)
		return pluginCheckResultMsg{Name: name, Outdated: outdated, Err: err}
	}
}

// installPluginCmd returns a tea.Cmd that clones a plugin.
func installPluginCmd(cloner git.Cloner, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CloneTimeout)
		defer cancel()

		err := git.CloneWithFallback(ctx, cloner, git.CloneOptions{
			URL:    op.Spec,
			Dir:    op.Path,
			Branch: op.Branch,
		}, plugin.NormalizeURL)

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
		ctx, cancel := context.WithTimeout(context.Background(), UpdateTimeout)
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

// removeDirCmd removes a directory and returns a message via msgFactory.
func removeDirCmd(op pendingOp, msgFactory func(name string, success bool, message string) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if err := os.RemoveAll(op.Path); err != nil {
			return msgFactory(op.Name, false, err.Error())
		}
		if _, statErr := os.Stat(op.Path); statErr == nil {
			return msgFactory(op.Name, false, "directory still exists after removal")
		}
		return msgFactory(op.Name, true, "removed successfully")
	}
}

// cleanPluginCmd returns a tea.Cmd that removes an orphaned directory.
func cleanPluginCmd(op pendingOp) tea.Cmd {
	return removeDirCmd(op, func(name string, success bool, message string) tea.Msg {
		return pluginCleanResultMsg{Name: name, Success: success, Message: message}
	})
}

// uninstallPluginCmd returns a tea.Cmd that removes an installed plugin directory.
func uninstallPluginCmd(op pendingOp) tea.Cmd {
	return removeDirCmd(op, func(name string, success bool, message string) tea.Msg {
		return pluginUninstallResultMsg{Name: name, Success: success, Message: message}
	})
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
		return installPluginCmd(m.deps.Cloner, op)
	case OpUpdate:
		return updatePluginCmd(m.deps.Puller, op)
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
		if p.Status.IsInstalled() {
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
			if p.Status.IsInstalled() {
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
		if p.Status.IsInstalled() {
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
