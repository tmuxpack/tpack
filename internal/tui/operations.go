package tui

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// Messages returned by operations.
type pluginInstallResultMsg struct {
	Name    string
	Success bool
	Message string
}

type pluginUpdateResultMsg struct {
	Name      string
	Success   bool
	Message   string
	Output    string
	Commits   []git.Commit
	Dir       string
	BeforeRef string
	AfterRef  string
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

type pluginRemoveResultMsg struct {
	Name    string
	Success bool
	Message string
}

type pluginCheckResultMsg struct {
	Name     string
	Outdated bool
	Err      error
}

// checks if a plugin is outdated
func checkPluginCmd(fetcher git.Fetcher, name string, dir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CheckTimeout)
		defer cancel()

		outdated, err := fetcher.IsOutdated(ctx, dir)
		return pluginCheckResultMsg{Name: name, Outdated: outdated, Err: err}
	}
}

// clones a plugin
func installPluginCmd(cloner git.Cloner, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CloneTimeout)
		defer cancel()

		err := git.CloneWithFallback(ctx, cloner, git.CloneOptions{
			URL:    op.Spec,
			Dir:    op.Path,
			Branch: op.Branch,
		}, plug.NormalizeURL)

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

// pulls updates
func updatePluginCmd(puller git.Puller, revParser git.RevParser, logger git.Logger, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), UpdateTimeout)
		defer cancel()

		// Capture HEAD before pull for commit log comparison.
		var beforeHash string
		if revParser != nil {
			beforeHash, _ = revParser.RevParse(ctx, op.Path)
		}

		output, err := puller.Pull(ctx, git.PullOptions{Dir: op.Path, Branch: op.Branch})
		if err != nil {
			return pluginUpdateResultMsg{
				Name:    op.Name,
				Success: false,
				Message: err.Error(),
				Output:  output,
			}
		}

		// Get commits pulled if we captured the before hash.
		var commits []git.Commit
		var afterHash string
		if beforeHash != "" && logger != nil {
			var revErr error
			afterHash, revErr = revParser.RevParse(ctx, op.Path)
			if revErr == nil && afterHash != beforeHash {
				commits, _ = logger.Log(ctx, op.Path, beforeHash, afterHash)
			}
		}

		return pluginUpdateResultMsg{
			Name:      op.Name,
			Success:   true,
			Message:   "updated successfully",
			Output:    output,
			Commits:   commits,
			Dir:       op.Path,
			BeforeRef: beforeHash,
			AfterRef:  afterHash,
		}
	}
}

func removeDirCmd(op pendingOp, msgFactory func(name string, success bool, message string) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if err := os.RemoveAll(op.Path); err != nil {
			return msgFactory(op.Name, false, err.Error())
		}
		return msgFactory(op.Name, true, "removed successfully")
	}
}

// removes orphaned directories
func cleanPluginCmd(op pendingOp) tea.Cmd {
	return removeDirCmd(op, func(name string, success bool, message string) tea.Msg {
		return pluginCleanResultMsg{Name: name, Success: success, Message: message}
	})
}

func uninstallPluginCmd(op pendingOp) tea.Cmd {
	return removeDirCmd(op, func(name string, success bool, message string) tea.Msg {
		return pluginUninstallResultMsg{Name: name, Success: success, Message: message}
	})
}

func removePluginDirCmd(op pendingOp) tea.Cmd {
	return removeDirCmd(op, func(name string, success bool, message string) tea.Msg {
		return pluginRemoveResultMsg{Name: name, Success: success, Message: message}
	})
}

// sources tmux config file
func sourceCmd(runner tmux.Runner, confPath string) tea.Cmd {
	return func() tea.Msg {
		err := runner.SourceFile(confPath)
		return sourceCompleteMsg{Err: err}
	}
}

// dispatches up to maxConcurrentOps pending operations concurrently.
// Returns nil if the queue is empty and no operations are in flight.
func (m *Model) dispatchNext() tea.Cmd {
	slots := maxConcurrentOps - m.inFlight
	if slots <= 0 {
		return nil
	}
	if len(m.pendingItems) == 0 {
		if m.inFlight == 0 {
			m.processing = false
			if m.deps.Runner != nil && (m.operation == OpInstall || m.operation == OpUpdate) {
				return sourceCmd(m.deps.Runner, m.cfg.TmuxConf)
			}
		}
		return nil
	}

	n := min(slots, len(m.pendingItems))
	batch := m.pendingItems[:n]
	m.pendingItems = m.pendingItems[n:]

	var cmds []tea.Cmd
	for _, op := range batch {
		m.inFlight++
		m.inFlightNames = append(m.inFlightNames, op.Name)

		switch m.operation {
		case OpNone:
			// No-op; should not reach here.
		case OpInstall:
			cmds = append(cmds, installPluginCmd(m.deps.Cloner, op))
		case OpRemove:
			cmds = append(cmds, removePluginDirCmd(op))
		case OpUpdate:
			cmds = append(cmds, updatePluginCmd(m.deps.Puller, m.deps.RevParser, m.deps.Logger, op))
		case OpClean:
			cmds = append(cmds, cleanPluginCmd(op))
		case OpUninstall:
			cmds = append(cmds, uninstallPluginCmd(op))
		}
	}

	if len(cmds) == 0 {
		m.processing = false
		return nil
	}
	return tea.Batch(cmds...)
}

// buildOpsFromTargeted builds pending operations from the targeted plugins (selected or cursor),
// filtered by the given predicate. Pass nil to include all targeted plugins.
func (m *Model) buildOpsFromTargeted(filter func(PluginItem) bool) []pendingOp {
	indices := m.targetIndices()
	var ops []pendingOp
	for _, i := range indices {
		p := m.plugins[i]
		if filter != nil && !filter(p) {
			continue
		}
		ops = append(ops, pendingOp{
			Name:   p.Name,
			Spec:   p.Spec,
			Branch: p.Branch,
			Path:   plug.PluginPath(p.Name, m.cfg.PluginPath),
		})
	}
	return ops
}

// buildOpsFromAll builds pending operations from all plugins matching the given predicate.
func (m *Model) buildOpsFromAll(filter func(PluginItem) bool) []pendingOp {
	var ops []pendingOp
	for _, p := range m.plugins {
		if !filter(p) {
			continue
		}
		ops = append(ops, pendingOp{
			Name:   p.Name,
			Spec:   p.Spec,
			Branch: p.Branch,
			Path:   plug.PluginPath(p.Name, m.cfg.PluginPath),
		})
	}
	return ops
}

func isNotInstalled(p PluginItem) bool { return p.Status == StatusNotInstalled }
func isInstalled(p PluginItem) bool    { return p.Status.IsInstalled() }

func (m *Model) buildInstallOps() []pendingOp {
	return m.buildOpsFromTargeted(isNotInstalled)
}

func (m *Model) buildRemoveOps() []pendingOp {
	return m.buildOpsFromTargeted(nil)
}

func (m *Model) buildUpdateOps() []pendingOp {
	ops := m.buildOpsFromTargeted(isInstalled)
	// If nothing selected and no cursor match, update all installed.
	if len(ops) == 0 && !m.multiSelectActive {
		ops = m.buildOpsFromAll(isInstalled)
	}
	return ops
}

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

func (m *Model) buildUninstallOps() []pendingOp {
	return m.buildOpsFromTargeted(isInstalled)
}

func (m *Model) buildAutoInstallOps() []pendingOp {
	return m.buildOpsFromAll(isNotInstalled)
}

func (m *Model) buildAutoUpdateOps() []pendingOp {
	return m.buildOpsFromAll(isInstalled)
}

// returns the indices to operate on: selected if any, else cursor.
func (m *Model) targetIndices() []int {
	if m.multiSelectActive {
		return m.selectedIndices()
	}
	if m.listScroll.cursor >= 0 && m.listScroll.cursor < len(m.plugins) {
		return []int{m.listScroll.cursor}
	}
	return nil
}
