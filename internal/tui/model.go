package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmux-plugins/tpm/internal/config"
	"github.com/tmux-plugins/tpm/internal/git"
	"github.com/tmux-plugins/tpm/internal/plugin"
	"github.com/tmux-plugins/tpm/internal/tmux"
)

// Deps groups the external dependencies needed by the TUI.
type Deps struct {
	Cloner    git.Cloner
	Puller    git.Puller
	Validator git.Validator
	Fetcher   git.Fetcher
	RevParser git.RevParser
	Logger    git.Logger
	Runner    tmux.Runner // optional, for post-op tmux sourcing
}

// ModelOption configures optional Model behavior.
type ModelOption func(*Model)

// WithAutoOp returns a ModelOption that auto-starts the given operation on init.
func WithAutoOp(op Operation) ModelOption {
	return func(m *Model) { m.autoOp = op }
}

// autoStartMsg is sent by Init when an auto-operation is configured.
type autoStartMsg struct{}

// sourceCompleteMsg is sent after tmux source-file completes.
type sourceCompleteMsg struct{ Err error }

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	cfg     *config.Config
	plugins []PluginItem
	orphans []OrphanItem
	deps    Deps

	screen    Screen
	operation Operation
	autoOp    Operation
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
	checkSpinner    spinner.Model

	resultCursor       int
	resultScrollOffset int

	commitViewName         string
	commitViewCommits      []git.Commit
	commitViewCursor       int
	commitViewScrollOffset int
}

// NewModel creates a new Model from the resolved config and gathered plugins.
func NewModel(cfg *config.Config, plugins []plugin.Plugin, deps Deps, opts ...ModelOption) Model {
	items := buildPluginItems(plugins, cfg.PluginPath, deps.Validator)
	orphans := findOrphans(plugins, cfg.PluginPath)

	s := spinner.New()
	s.Spinner = spinner.Dot

	m := Model{
		cfg:          cfg,
		plugins:      items,
		orphans:      orphans,
		deps:         deps,
		screen:       ScreenList,
		operation:    OpNone,
		selected:     make(map[int]bool),
		progressBar:  newProgress(),
		checkSpinner: s,
		width:        FixedWidth,
		height:       FixedHeight,
		sizeKnown:    true,
	}
	m.viewHeight = FixedHeight - TitleReservedLines
	if m.viewHeight < MinViewHeight {
		m.viewHeight = MinViewHeight
	}
	m.progressBar.Width = FixedWidth - ProgressBarPadding
	if m.progressBar.Width > ProgressBarMaxWidth {
		m.progressBar.Width = ProgressBarMaxWidth
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.plugins {
		if p.Status == StatusChecking {
			dir := plugin.PluginPath(p.Name, m.cfg.PluginPath)
			cmds = append(cmds, checkPluginCmd(m.deps.Fetcher, p.Name, dir))
		}
	}
	if len(cmds) > 0 {
		cmds = append(cmds, m.checkSpinner.Tick)
	}
	if m.autoOp != OpNone {
		cmds = append(cmds, func() tea.Msg { return autoStartMsg{} })
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
// Bubble Tea requires a value receiver so the framework can manage immutable
// model snapshots; mutations are returned as new values via (tea.Model, tea.Cmd).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		if pm, ok := progressModel.(progress.Model); ok {
			m.progressBar = pm
		}
		return m, cmd

	case autoStartMsg:
		return m.startAutoOperation()
	case sourceCompleteMsg:
		return m, nil
	case pluginCheckResultMsg:
		return m.handleCheckResult(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
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

// handleWindowSize updates layout dimensions from the actual terminal/popup size.
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.sizeKnown = true
	m.viewHeight = msg.Height - TitleReservedLines
	if m.viewHeight < MinViewHeight {
		m.viewHeight = MinViewHeight
	}
	m.progressBar.Width = msg.Width - ProgressBarPadding
	if m.progressBar.Width > ProgressBarMaxWidth {
		m.progressBar.Width = ProgressBarMaxWidth
	}
}

// handleKeyMsg routes key events to the appropriate screen handler.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, SharedKeys.ForceQuit) {
		return m, tea.Quit
	}
	switch m.screen {
	case ScreenCommits:
		return m.updateCommitView(msg)
	case ScreenProgress:
		return m.updateProgress(msg)
	case ScreenList:
		return m.updateList(msg)
	}
	return m.updateList(msg)
}

// View implements tea.Model.
func (m Model) View() string {
	var content string
	switch m.screen {
	case ScreenList:
		content = m.viewList()
	case ScreenProgress:
		content = m.viewProgress()
	case ScreenCommits:
		content = m.viewCommits()
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

	if m.autoOp != OpNone {
		switch {
		case key.Matches(msg, SharedKeys.Quit), msg.String() == escKeyName:
			return m, tea.Quit
		case key.Matches(msg, ListKeys.Up):
			m.moveResultCursorUp()
		case key.Matches(msg, ListKeys.Down):
			m.moveResultCursorDown()
		case msg.String() == "enter":
			m.showCommits()
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case msg.String() == escKeyName:
		return m.returnToList(), nil
	case key.Matches(msg, ListKeys.Up):
		m.moveResultCursorUp()
	case key.Matches(msg, ListKeys.Down):
		m.moveResultCursorDown()
	case msg.String() == "enter":
		m.showCommits()
	}
	return m, nil
}

// resultMaxVisible returns the number of result rows that fit in the current height.
func (m *Model) resultMaxVisible() int {
	v := m.height - progressResultsReservedLines
	if v < MinViewHeight {
		return MinViewHeight
	}
	return v
}

// moveResultCursorUp moves the result cursor up and adjusts scroll.
func (m *Model) moveResultCursorUp() {
	if m.resultCursor > 0 {
		m.resultCursor--
		if m.resultCursor < m.resultScrollOffset+ScrollOffsetMargin && m.resultScrollOffset > 0 {
			m.resultScrollOffset--
		}
	}
}

// moveResultCursorDown moves the result cursor down and adjusts scroll.
func (m *Model) moveResultCursorDown() {
	if m.resultCursor < len(m.results)-1 {
		m.resultCursor++
		if m.resultCursor >= m.resultScrollOffset+m.resultMaxVisible()-ScrollOffsetMargin {
			maxOffset := len(m.results) - m.resultMaxVisible()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.resultScrollOffset < maxOffset {
				m.resultScrollOffset++
			}
		}
	}
}

// showCommits navigates to the inline commit viewer for the current result.
func (m *Model) showCommits() bool {
	if m.resultCursor < 0 || m.resultCursor >= len(m.results) {
		return false
	}
	r := m.results[m.resultCursor]
	if len(r.Commits) == 0 {
		return false
	}
	m.screen = ScreenCommits
	m.commitViewName = r.Name
	m.commitViewCommits = r.Commits
	m.commitViewCursor = 0
	m.commitViewScrollOffset = 0
	return true
}

// returnToProgress transitions back to the progress screen from the commit viewer.
func (m *Model) returnToProgress() {
	m.screen = ScreenProgress
	m.commitViewName = ""
	m.commitViewCommits = nil
	m.commitViewCursor = 0
	m.commitViewScrollOffset = 0
}

// updateCommitView handles key events on the commit viewer screen.
func (m Model) updateCommitView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SharedKeys.Quit), msg.String() == escKeyName:
		m.returnToProgress()
	case key.Matches(msg, ListKeys.Up):
		m.moveCommitCursorUp()
	case key.Matches(msg, ListKeys.Down):
		m.moveCommitCursorDown()
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
	m.resultCursor = 0
	m.resultScrollOffset = 0

	cmd := m.dispatchNext()
	return m, cmd
}

// startAutoOperation transitions to the progress screen for an auto-triggered
// operation, targeting all applicable plugins rather than the cursor/selection.
func (m Model) startAutoOperation() (tea.Model, tea.Cmd) {
	var ops []pendingOp
	switch m.autoOp {
	case OpNone, OpUninstall:
		return m, nil
	case OpInstall:
		ops = m.buildAutoInstallOps()
	case OpUpdate:
		ops = m.buildAutoUpdateOps()
	case OpClean:
		ops = m.buildCleanOps()
	}

	if len(ops) == 0 {
		return m, tea.Quit
	}

	m.screen = ScreenProgress
	m.operation = m.autoOp
	m.pendingItems = ops
	m.totalItems = len(ops)
	m.completedItems = 0
	m.results = nil
	m.processing = true
	m.currentItemName = ""
	m.resultCursor = 0
	m.resultScrollOffset = 0

	cmd := m.dispatchNext()
	return m, cmd
}

// handleOpResult is the shared logic for processing an operation result:
// increment counter, append result, optionally update plugin status, dispatch next.
func (m *Model) handleOpResult(name string, success bool, message string, updateStatus func()) tea.Cmd {
	m.completedItems++
	m.results = append(m.results, ResultItem{
		Name:    name,
		Success: success,
		Message: message,
	})
	if success && updateStatus != nil {
		updateStatus()
	}
	return m.dispatchNext()
}

// handleInstallResult processes an install result and dispatches next.
func (m Model) handleInstallResult(msg pluginInstallResultMsg) (tea.Model, tea.Cmd) {
	cmd := m.handleOpResult(msg.Name, msg.Success, msg.Message, func() {
		for i := range m.plugins {
			if m.plugins[i].Name == msg.Name {
				m.plugins[i].Status = StatusInstalled
				break
			}
		}
	})
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
	cmd := m.handleOpResult(msg.Name, msg.Success, msg.Message, nil)
	return m, cmd
}

// handleUninstallResult processes an uninstall result and dispatches next.
func (m Model) handleUninstallResult(msg pluginUninstallResultMsg) (tea.Model, tea.Cmd) {
	cmd := m.handleOpResult(msg.Name, msg.Success, msg.Message, func() {
		for i := range m.plugins {
			if m.plugins[i].Name == msg.Name {
				m.plugins[i].Status = StatusNotInstalled
				break
			}
		}
	})
	return m, cmd
}

// handleSpinnerTick advances the spinner animation while checks are in progress.
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.checkSpinner, cmd = m.checkSpinner.Update(msg)
	if !m.hasCheckingPlugins() {
		return m, nil
	}
	return m, cmd
}

// handleCheckResult processes a background update check result.
func (m Model) handleCheckResult(msg pluginCheckResultMsg) (tea.Model, tea.Cmd) {
	for i := range m.plugins {
		if m.plugins[i].Name == msg.Name {
			switch {
			case msg.Err != nil:
				m.plugins[i].Status = StatusCheckFailed
			case msg.Outdated:
				m.plugins[i].Status = StatusOutdated
			default:
				m.plugins[i].Status = StatusInstalled
			}
			break
		}
	}
	return m, nil
}

// hasCheckingPlugins returns true if any plugin is still being checked.
func (m *Model) hasCheckingPlugins() bool {
	for _, p := range m.plugins {
		if p.Status == StatusChecking {
			return true
		}
	}
	return false
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
	m.resultCursor = 0
	m.resultScrollOffset = 0

	// Clamp cursor.
	if m.cursor >= len(m.plugins) {
		m.cursor = len(m.plugins) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}
