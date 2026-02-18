package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/git"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/registry"
	"github.com/tmuxpack/tpack/internal/tmux"
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

// WithTheme returns a ModelOption that sets the theme.
func WithTheme(t Theme) ModelOption {
	return func(m *Model) { m.theme = t }
}

// WithVersion returns a ModelOption that sets the version string.
func WithVersion(v string) ModelOption {
	return func(m *Model) { m.version = v }
}

// WithBinaryPath returns a ModelOption that sets the binary path.
func WithBinaryPath(p string) ModelOption {
	return func(m *Model) { m.binaryPath = p }
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
	theme   Theme

	screen    Screen
	operation Operation
	autoOp    Operation

	listScroll scrollState
	width      int
	height     int
	viewHeight int
	sizeKnown  bool

	selected          map[int]bool
	multiSelectActive bool

	results        []ResultItem
	totalItems     int
	completedItems int
	inFlightNames  []string
	processing     bool
	inFlight       int
	pendingItems   []pendingOp
	progressBar    progress.Model
	checkSpinner   spinner.Model

	resultScroll scrollState

	commitViewName    string
	commitViewCommits []git.Commit
	commitScroll      scrollState

	// Browse screen state.
	browseRegistry      *registry.Registry
	browseResults       []registry.RegistryItem
	browseQuery         string
	browseQuerySnapshot string
	browseInput         textinput.Model
	browseCategory      int // index into registry.Categories, -1 = all
	browseScroll        scrollState
	browseLoading       bool
	browseErr           error
	browseStatus        string
	searching           bool

	version    string
	binaryPath string
}

// NewModel creates a new Model from the resolved config and gathered plugins.
func NewModel(cfg *config.Config, plugins []plug.Plugin, deps Deps, opts ...ModelOption) Model {
	items := buildPluginItems(plugins, cfg.PluginPath, deps.Validator)
	orphans := findOrphans(plugins, cfg.PluginPath)

	s := spinner.New()
	s.Spinner = spinner.Dot

	m := Model{
		cfg:          cfg,
		plugins:      items,
		orphans:      orphans,
		deps:         deps,
		theme:        DefaultTheme(),
		screen:       ScreenList,
		operation:    OpNone,
		selected:     make(map[int]bool),
		progressBar:  newProgress(),
		checkSpinner: s,
		width:        FixedWidth,
		height:       FixedHeight,
		sizeKnown:    true,
	}
	m.viewHeight = max(FixedHeight-TitleReservedLines, MinViewHeight)
	ti := textinput.New()
	ti.Placeholder = "Filter plugins..."
	ti.CharLimit = 100
	m.browseInput = ti
	m.browseCategory = -1
	m.progressBar.Width = min(FixedWidth-ProgressBarPadding, ProgressBarMaxWidth)
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
			dir := plug.PluginPath(p.Name, m.cfg.PluginPath)
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
	case registryFetchResultMsg:
		return m.handleRegistryFetch(msg)
	case clearBrowseStatusMsg:
		m.browseStatus = ""
		return m, nil
	}

	return m, nil
}

// handleWindowSize updates layout dimensions from the actual terminal/popup size.
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.sizeKnown = true
	m.viewHeight = max(msg.Height-TitleReservedLines, MinViewHeight)
	m.progressBar.Width = min(msg.Width-ProgressBarPadding, ProgressBarMaxWidth)
}

// handleKeyMsg routes key events to the appropriate screen handler.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, SharedKeys.ForceQuit) {
		return m, tea.Quit
	}
	switch m.screen {
	case ScreenCommits:
		return m.handleKeyMsgCommit(msg)
	case ScreenProgress:
		return m.handleKeyMsgProgress(msg)
	case ScreenDebug:
		return m.handleKeyMsgDebug(msg)
	case ScreenBrowse:
		return m.handleKeyMsgBrowse(msg)
	case ScreenList:
		return m.handleKeyMsgList(msg)
	}
	return m.handleKeyMsgList(msg)
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
	case ScreenDebug:
		content = m.viewDebug()
	case ScreenBrowse:
		content = m.viewBrowse()
	}
	return m.theme.BaseStyle.Render(content)
}

// handleKeyMsgList handles key events on the list screen.
func (m Model) handleKeyMsgList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case key.Matches(msg, ListKeys.Up):
		m.listScroll.moveUp()
	case key.Matches(msg, ListKeys.Down):
		m.listScroll.moveDown(len(m.plugins), m.viewHeight)
	case key.Matches(msg, ListKeys.Toggle):
		if len(m.plugins) > 0 {
			m.toggleSelection(m.listScroll.cursor)
		}
	case key.Matches(msg, ListKeys.Install):
		return m.startOperation(OpInstall)
	case key.Matches(msg, ListKeys.Update):
		return m.startOperation(OpUpdate)
	case key.Matches(msg, ListKeys.Clean):
		return m.startOperation(OpClean)
	case key.Matches(msg, ListKeys.Uninstall):
		return m.startOperation(OpUninstall)
	case key.Matches(msg, ListKeys.Browse):
		return m.enterBrowse()
	case key.Matches(msg, ListKeys.Debug):
		m.screen = ScreenDebug
	}
	return m, nil
}

// handleKeyMsgProgress handles key events on the progress screen.
func (m Model) handleKeyMsgProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.processing {
		return m, nil
	}

	visible := m.displayResults()

	if m.autoOp != OpNone {
		switch {
		case key.Matches(msg, SharedKeys.Quit), msg.String() == escKeyName:
			return m, tea.Quit
		case key.Matches(msg, ListKeys.Up):
			m.resultScroll.moveUp()
		case key.Matches(msg, ListKeys.Down):
			m.resultScroll.moveDown(len(visible), m.resultMaxVisible())
		case msg.String() == "enter":
			m.showCommitsFromVisible(visible)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case msg.String() == escKeyName:
		return m.returnToList(), nil
	case key.Matches(msg, ListKeys.Up):
		m.resultScroll.moveUp()
	case key.Matches(msg, ListKeys.Down):
		m.resultScroll.moveDown(len(visible), m.resultMaxVisible())
	case msg.String() == "enter":
		m.showCommitsFromVisible(visible)
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

// showCommitsFromVisible navigates to the commit viewer using the given visible results slice.
func (m *Model) showCommitsFromVisible(visible []ResultItem) bool {
	if m.resultScroll.cursor < 0 || m.resultScroll.cursor >= len(visible) {
		return false
	}
	r := visible[m.resultScroll.cursor]
	if len(r.Commits) == 0 {
		return false
	}
	m.screen = ScreenCommits
	m.commitViewName = r.Name
	m.commitViewCommits = r.Commits
	m.commitScroll.reset()
	return true
}

// showCommits navigates to the inline commit viewer for the current result.
func (m *Model) showCommits() bool {
	return m.showCommitsFromVisible(m.displayResults())
}

// returnToProgress transitions back to the progress screen from the commit viewer.
func (m *Model) returnToProgress() {
	m.screen = ScreenProgress
	m.commitViewName = ""
	m.commitViewCommits = nil
	m.commitScroll.reset()
}

// handleKeyMsgCommit handles key events on the commit viewer screen.
func (m Model) handleKeyMsgCommit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SharedKeys.Quit), msg.String() == escKeyName:
		m.returnToProgress()
	case key.Matches(msg, ListKeys.Up):
		m.commitScroll.moveUp()
	case key.Matches(msg, ListKeys.Down):
		m.commitScroll.moveDown(len(m.commitViewCommits), m.commitMaxVisible())
	}
	return m, nil
}

// handleKeyMsgDebug handles key events on the debug screen.
func (m Model) handleKeyMsgDebug(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case msg.String() == escKeyName:
		m.screen = ScreenList
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
	m.inFlight = 0
	m.inFlightNames = nil
	m.resultScroll.reset()

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
	m.inFlight = 0
	m.inFlightNames = nil
	m.resultScroll.reset()

	cmd := m.dispatchNext()
	return m, cmd
}

// handleOpResult is the shared logic for processing an operation result:
// increment counter, append result, optionally update plugin status, dispatch next.
func (m *Model) handleOpResult(result ResultItem, updateStatus func()) tea.Cmd {
	m.completedItems++
	m.inFlight--
	m.results = append(m.results, result)
	if result.Success && updateStatus != nil {
		updateStatus()
	}
	// Remove from inFlightNames.
	for i, name := range m.inFlightNames {
		if name == result.Name {
			m.inFlightNames = append(m.inFlightNames[:i], m.inFlightNames[i+1:]...)
			break
		}
	}
	return m.dispatchNext()
}

// handleInstallResult processes an install result and dispatches next.
func (m Model) handleInstallResult(msg pluginInstallResultMsg) (tea.Model, tea.Cmd) {
	cmd := m.handleOpResult(ResultItem{Name: msg.Name, Success: msg.Success, Message: msg.Message}, func() {
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
	cmd := m.handleOpResult(ResultItem(msg), func() {
		for i := range m.plugins {
			if m.plugins[i].Name == msg.Name {
				m.plugins[i].Status = StatusInstalled
				break
			}
		}
	})
	return m, cmd
}

// handleCleanResult processes a clean result and dispatches next.
func (m Model) handleCleanResult(msg pluginCleanResultMsg) (tea.Model, tea.Cmd) {
	cmd := m.handleOpResult(ResultItem{Name: msg.Name, Success: msg.Success, Message: msg.Message}, nil)
	return m, cmd
}

// handleUninstallResult processes an uninstall result and dispatches next.
func (m Model) handleUninstallResult(msg pluginUninstallResultMsg) (tea.Model, tea.Cmd) {
	cmd := m.handleOpResult(ResultItem{Name: msg.Name, Success: msg.Success, Message: msg.Message}, func() {
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
	m.inFlight = 0
	m.inFlightNames = nil
	m.resultScroll.reset()

	// Clamp cursor.
	if m.listScroll.cursor >= len(m.plugins) {
		m.listScroll.cursor = len(m.plugins) - 1
	}
	if m.listScroll.cursor < 0 {
		m.listScroll.cursor = 0
	}

	return m
}
