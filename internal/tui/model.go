package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
)

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	cfg       *config.Config
	plugins   []PluginItem
	orphans   []OrphanItem
	cloner    git.Cloner
	puller    git.Puller
	validator git.Validator

	screen    Screen
	operation Operation
	cursor    int

	scrollOffset int
	width        int
	height       int
	viewHeight   int
	sizeKnown    bool

	selected          map[int]bool
	multiSelectActive bool

	results         []ResultItem
	totalItems      int
	completedItems  int
	currentItemName string
	processing      bool
	pendingItems    []pendingOp
	progressBar     progress.Model
}

// NewModel creates a new Model from the resolved config and gathered plugins.
func NewModel(
	cfg *config.Config,
	plugins []plugin.Plugin,
	cloner git.Cloner,
	puller git.Puller,
	validator git.Validator,
) Model {
	items := buildPluginItems(plugins, cfg.PluginPath, validator)
	orphans := findOrphans(plugins, cfg.PluginPath)

	return Model{
		cfg:         cfg,
		plugins:     items,
		orphans:     orphans,
		cloner:      cloner,
		puller:      puller,
		validator:   validator,
		screen:      ScreenList,
		operation:   OpNone,
		selected:    make(map[int]bool),
		progressBar: newProgress(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sizeKnown = true
		// Reserve lines for title, subtitle, header, help, padding.
		m.viewHeight = msg.Height - 12
		if m.viewHeight < 3 {
			m.viewHeight = 3
		}
		m.progressBar.Width = msg.Width - 8
		if m.progressBar.Width > 60 {
			m.progressBar.Width = 60
		}
		return m, nil

	case tea.KeyMsg:
		// Force quit always works.
		if key.Matches(msg, SharedKeys.ForceQuit) {
			return m, tea.Quit
		}

		if m.screen == ScreenProgress {
			return m.updateProgress(msg)
		}
		return m.updateList(msg)

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		if pm, ok := progressModel.(progress.Model); ok {
			m.progressBar = pm
		}
		return m, cmd

	case pluginInstallResultMsg:
		return m.handleInstallResult(msg)
	case pluginUpdateResultMsg:
		return m.handleUpdateResult(msg)
	case pluginCleanResultMsg:
		return m.handleCleanResult(msg)
	case pluginUninstallResultMsg:
		return m.handleUninstallResult(msg)
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var content string
	switch m.screen {
	case ScreenList:
		content = m.viewList()
	case ScreenProgress:
		content = m.viewProgress()
	}
	return BaseStyle.Render(content)
}

// updateList handles key events on the list screen.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case key.Matches(msg, ListKeys.Up):
		m.moveCursorUp()
	case key.Matches(msg, ListKeys.Down):
		m.moveCursorDown()
	case key.Matches(msg, ListKeys.Toggle):
		if len(m.plugins) > 0 {
			m.toggleSelection(m.cursor)
		}
	case key.Matches(msg, ListKeys.Install):
		return m.startOperation(OpInstall)
	case key.Matches(msg, ListKeys.Update):
		return m.startOperation(OpUpdate)
	case key.Matches(msg, ListKeys.Clean):
		return m.startOperation(OpClean)
	case key.Matches(msg, ListKeys.Uninstall):
		return m.startOperation(OpUninstall)
	}
	return m, nil
}

// moveCursorUp moves the cursor up and adjusts scroll.
func (m *Model) moveCursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.scrollOffset+ScrollOffsetMargin && m.scrollOffset > 0 {
			m.scrollOffset--
		}
	}
}

// moveCursorDown moves the cursor down and adjusts scroll.
func (m *Model) moveCursorDown() {
	if m.cursor < len(m.plugins)-1 {
		m.cursor++
		if m.cursor >= m.scrollOffset+m.viewHeight-ScrollOffsetMargin {
			maxOffset := len(m.plugins) - m.viewHeight
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.scrollOffset < maxOffset {
				m.scrollOffset++
			}
		}
	}
}

// updateProgress handles key events on the progress screen.
func (m Model) updateProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.processing {
		return m, nil
	}

	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case msg.String() == "esc":
		return m.returnToList(), nil
	}
	return m, nil
}

// startOperation transitions to the progress screen and begins an operation.
func (m Model) startOperation(op Operation) (tea.Model, tea.Cmd) {
	var ops []pendingOp
	switch op {
	case OpNone:
		return m, nil
	case OpInstall:
		ops = m.buildInstallOps()
	case OpUpdate:
		ops = m.buildUpdateOps()
	case OpClean:
		ops = m.buildCleanOps()
	case OpUninstall:
		ops = m.buildUninstallOps()
	}

	if len(ops) == 0 {
		return m, nil
	}

	m.screen = ScreenProgress
	m.operation = op
	m.pendingItems = ops
	m.totalItems = len(ops)
	m.completedItems = 0
	m.results = nil
	m.processing = true
	m.currentItemName = ""

	cmd := m.dispatchNext()
	return m, cmd
}

// handleInstallResult processes an install result and dispatches next.
func (m Model) handleInstallResult(msg pluginInstallResultMsg) (tea.Model, tea.Cmd) {
	m.completedItems++
	m.results = append(m.results, ResultItem{
		Name:    msg.Name,
		Success: msg.Success,
		Message: msg.Message,
	})

	// Update plugin status if successful.
	if msg.Success {
		for i := range m.plugins {
			if m.plugins[i].Name == msg.Name {
				m.plugins[i].Status = StatusInstalled
				break
			}
		}
	}

	cmd := m.dispatchNext()
	return m, cmd
}

// handleUpdateResult processes an update result and dispatches next.
func (m Model) handleUpdateResult(msg pluginUpdateResultMsg) (tea.Model, tea.Cmd) {
	m.completedItems++
	m.results = append(m.results, ResultItem(msg))

	cmd := m.dispatchNext()
	return m, cmd
}

// handleCleanResult processes a clean result and dispatches next.
func (m Model) handleCleanResult(msg pluginCleanResultMsg) (tea.Model, tea.Cmd) {
	m.completedItems++
	m.results = append(m.results, ResultItem{
		Name:    msg.Name,
		Success: msg.Success,
		Message: msg.Message,
	})

	cmd := m.dispatchNext()
	return m, cmd
}

// handleUninstallResult processes an uninstall result and dispatches next.
func (m Model) handleUninstallResult(msg pluginUninstallResultMsg) (tea.Model, tea.Cmd) {
	m.completedItems++
	m.results = append(m.results, ResultItem{
		Name:    msg.Name,
		Success: msg.Success,
		Message: msg.Message,
	})

	if msg.Success {
		for i := range m.plugins {
			if m.plugins[i].Name == msg.Name {
				m.plugins[i].Status = StatusNotInstalled
				break
			}
		}
	}

	cmd := m.dispatchNext()
	return m, cmd
}

// returnToList transitions back to the list screen and refreshes state.
func (m Model) returnToList() Model {
	m.screen = ScreenList
	m.operation = OpNone
	m.processing = false
	m.clearSelection()

	// Remove cleaned orphans from the list.
	if len(m.results) > 0 {
		removedSet := make(map[string]bool)
		for _, r := range m.results {
			if r.Success {
				removedSet[r.Name] = true
			}
		}
		var remaining []OrphanItem
		for _, o := range m.orphans {
			if !removedSet[o.Name] {
				remaining = append(remaining, o)
			}
		}
		m.orphans = remaining
	}

	m.results = nil
	m.pendingItems = nil

	// Clamp cursor.
	if m.cursor >= len(m.plugins) {
		m.cursor = len(m.plugins) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}
