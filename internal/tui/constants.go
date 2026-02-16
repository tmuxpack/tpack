package tui

import (
	"time"

	"github.com/tmux-plugins/tpm/internal/git"
)

// Screen represents the current TUI screen.
type Screen int

const (
	ScreenList Screen = iota
	ScreenProgress
	ScreenCommits
	ScreenDebug
)

// Operation represents the current plugin operation.
type Operation int

const (
	OpNone Operation = iota
	OpInstall
	OpUpdate
	OpClean
	OpUninstall
)

func (o Operation) String() string {
	switch o {
	case OpNone:
		return ""
	case OpInstall:
		return "Install"
	case OpUpdate:
		return "Update"
	case OpClean:
		return "Clean"
	case OpUninstall:
		return "Uninstall"
	default:
		return ""
	}
}

// PluginStatus represents the install status of a plugin.
type PluginStatus int

const (
	StatusInstalled PluginStatus = iota
	StatusNotInstalled
	StatusChecking
	StatusOutdated
	StatusCheckFailed
)

// IsInstalled returns true for any status that means the plugin is on disk.
func (s PluginStatus) IsInstalled() bool {
	switch s {
	case StatusInstalled, StatusChecking, StatusOutdated, StatusCheckFailed:
		return true
	case StatusNotInstalled:
		return false
	}
	return false
}

func (s PluginStatus) String() string {
	switch s {
	case StatusInstalled:
		return "Installed"
	case StatusNotInstalled:
		return "Not Installed"
	case StatusChecking:
		return "Checking"
	case StatusOutdated:
		return "Outdated"
	case StatusCheckFailed:
		return "Check Failed"
	default:
		return "Unknown"
	}
}

// PluginItem is an enriched plugin with install status.
type PluginItem struct {
	Name   string
	Spec   string
	Branch string
	Status PluginStatus
}

// OrphanItem represents a plugin directory not in config.
type OrphanItem struct {
	Name string
	Path string
}

// ResultItem holds the result of a single operation.
type ResultItem struct {
	Name      string
	Success   bool
	Message   string
	Output    string
	Commits   []git.Commit
	Dir       string
	BeforeRef string
	AfterRef  string
}

// pendingOp is a queued operation item.
type pendingOp struct {
	Name   string
	Spec   string
	Branch string
	Path   string
}

// escKeyName is the string representation of the Escape key.
const escKeyName = "esc"

// Fixed application dimensions.
const (
	// FixedWidth is the fixed width of the application in columns.
	FixedWidth = 80
	// FixedHeight is the fixed height of the application in rows.
	FixedHeight = 25
)

// Layout constants.
const (
	// ScrollOffsetMargin is the number of rows to keep visible above/below cursor.
	ScrollOffsetMargin = 3
	// TitleReservedLines is the number of lines reserved for title/subtitle/header/help/padding.
	TitleReservedLines = 12
	// MinViewHeight is the minimum number of visible plugin rows.
	MinViewHeight = 3
	// ProgressBarMaxWidth is the maximum width of the progress bar.
	ProgressBarMaxWidth = 60
	// ProgressBarPadding is the horizontal padding subtracted from terminal width for the progress bar.
	ProgressBarPadding = 8
	// BaseStylePadding is the total horizontal padding applied by BaseStyle (2 left + 2 right).
	BaseStylePadding = 4
	// BaseStyleVerticalPadding is the total vertical padding applied by BaseStyle (1 top + 1 bottom).
	BaseStyleVerticalPadding = 2
	// StatusColWidth is the approximate width of the status column.
	StatusColWidth = 14
	// progressResultsReservedLines is the overhead for title, counter, progress bar, stats, and help
	// on the progress screen.
	progressResultsReservedLines = 15
	// maxConcurrentOps is the maximum number of plugin operations dispatched concurrently.
	maxConcurrentOps = 3
)

// Timeout constants.
const (
	CheckTimeout  = 15 * time.Second
	CloneTimeout  = 2 * time.Minute
	UpdateTimeout = 2 * time.Minute
)
