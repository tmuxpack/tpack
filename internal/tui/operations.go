package tui

import (
	"context"
	"os"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/tmux"
)

// Messages returned by operations.
type operationResultMsg struct {
	Operation Operation
	ResultItem
}

type pluginCheckResultMsg struct {
	Name     string
	DirName  string
	Outdated bool
	Err      error
}

// checks if a plugin is outdated
func checkPluginCmd(fetcher git.Fetcher, name, dirName, dir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CheckTimeout)
		defer cancel()

		outdated, err := fetcher.IsOutdated(ctx, dir)
		return pluginCheckResultMsg{Name: name, DirName: dirName, Outdated: outdated, Err: err}
	}
}

// clones a plugin
func installPluginCmd(cloner git.Cloner, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CloneTimeout)
		defer cancel()

		var (
			warnMu   sync.Mutex
			warnings []string
		)
		err := git.CloneWithFallback(ctx, cloner, git.CloneOptions{
			URL:    op.Spec,
			Dir:    op.Path,
			Branch: op.Branch,
			OnWarning: func(msg string) {
				warnMu.Lock()
				defer warnMu.Unlock()
				warnings = append(warnings, msg)
			},
		}, plug.NormalizeURL)

		if err != nil {
			return operationResultMsg{
				Operation: OpInstall,
				ResultItem: ResultItem{
					Name: op.Name, DirName: op.DirName, Success: false, Message: err.Error(),
				},
			}
		}
		return operationResultMsg{
			Operation: OpInstall,
			ResultItem: ResultItem{
				Name: op.Name, DirName: op.DirName, Success: true,
				Message: appendWarnings("installed successfully", warnings),
			},
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

		var (
			warnMu   sync.Mutex
			warnings []string
		)
		output, err := puller.Pull(ctx, git.PullOptions{
			Dir:    op.Path,
			Branch: op.Branch,
			OnWarning: func(msg string) {
				warnMu.Lock()
				defer warnMu.Unlock()
				warnings = append(warnings, msg)
			},
		})
		if err != nil {
			return operationResultMsg{
				Operation: OpUpdate,
				ResultItem: ResultItem{
					Name: op.Name, DirName: op.DirName, Success: false, Message: err.Error(), Output: output,
				},
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

		return operationResultMsg{
			Operation: OpUpdate,
			ResultItem: ResultItem{
				Name:      op.Name,
				DirName:   op.DirName,
				Success:   true,
				Message:   appendWarnings("updated successfully", warnings),
				Output:    output,
				Commits:   commits,
				Dir:       op.Path,
				BeforeRef: beforeHash,
				AfterRef:  afterHash,
			},
		}
	}
}

// appendWarnings tacks any non-fatal warnings onto a result message so the
// TUI's existing rendering displays them alongside the success status.
func appendWarnings(base string, warnings []string) string {
	if len(warnings) == 0 {
		return base
	}
	return base + " (with warnings: " + strings.Join(warnings, "; ") + ")"
}

func removeDirCmd(operation Operation, op pendingOp) tea.Cmd {
	return func() tea.Msg {
		result := func(success bool, message string) operationResultMsg {
			return operationResultMsg{
				Operation: operation,
				ResultItem: ResultItem{
					Name: op.Name, DirName: op.DirName, Success: success, Message: message,
				},
			}
		}
		path := op.Path
		if operation != OpClean {
			root, err := op.Root.Resolved()
			if err != nil {
				return result(false, err.Error())
			}
			path, err = root.Child(op.DirName)
			if err != nil {
				return result(false, err.Error())
			}
		}
		if err := os.RemoveAll(path); err != nil {
			return result(false, err.Error())
		}
		return result(true, "removed successfully")
	}
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
		case OpUpdate:
			cmds = append(cmds, updatePluginCmd(m.deps.Puller, m.deps.RevParser, m.deps.Logger, op))
		case OpRemove, OpClean, OpUninstall:
			cmds = append(cmds, removeDirCmd(m.operation, op))
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
	return m.buildOpsFromTargetedWithRoot(filter, false)
}

func (m *Model) buildOpsFromTargetedWithRoot(filter func(PluginItem) bool, resolveRoot bool) []pendingOp {
	indices := m.targetIndices()
	var ops []pendingOp
	for _, i := range indices {
		p := m.plugins[i]
		if filter != nil && !filter(p) {
			continue
		}
		root := m.cfg.PluginPath
		pathRoot := root
		var err error
		if resolveRoot {
			pathRoot, err = root.Resolved()
		}
		var path string
		if err == nil {
			path, err = pathRoot.Child(p.DirName)
		}
		if err != nil {
			m.plugins[i].Status = StatusLoadFailed
			m.plugins[i].LoadErr = err.Error()
			continue
		}
		ops = append(ops, pendingOp{
			Raw:     p.Raw,
			Name:    p.Name,
			DirName: p.DirName,
			Spec:    p.Spec,
			Branch:  p.Branch,
			Path:    path,
			Root:    root,
		})
	}
	return ops
}

// buildOpsFromAll builds pending operations from all plugins matching the given predicate.
func (m *Model) buildOpsFromAll(filter func(PluginItem) bool) []pendingOp {
	var ops []pendingOp
	for i, p := range m.plugins {
		if !filter(p) {
			continue
		}
		path, err := m.cfg.PluginPath.Child(p.DirName)
		if err != nil {
			m.plugins[i].Status = StatusLoadFailed
			m.plugins[i].LoadErr = err.Error()
			continue
		}
		ops = append(ops, pendingOp{
			Raw:     p.Raw,
			Name:    p.Name,
			DirName: p.DirName,
			Spec:    p.Spec,
			Branch:  p.Branch,
			Path:    path,
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
	return m.buildOpsFromTargetedWithRoot(nil, true)
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
			Name:    o.Name,
			DirName: o.DirName,
			Path:    o.Path,
		})
	}
	return ops
}

func (m *Model) buildUninstallOps() []pendingOp {
	return m.buildOpsFromTargetedWithRoot(isInstalled, true)
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
