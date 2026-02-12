package tui

// Screen represents the current TUI screen.
type Screen int

const (
	ScreenList Screen = iota
	ScreenProgress
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
	}
	return ""
}

// PluginStatus represents the install status of a plugin.
type PluginStatus int

const (
	StatusInstalled PluginStatus = iota
	StatusNotInstalled
)

func (s PluginStatus) String() string {
	switch s {
	case StatusInstalled:
		return "Installed"
	case StatusNotInstalled:
		return "Not Installed"
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
	Name    string
	Success bool
	Message string
	Output  string
}

// pendingOp is a queued operation item.
type pendingOp struct {
	Name   string
	Spec   string
	Branch string
	Path   string
}

// ScrollOffsetMargin is the number of rows to keep visible above/below cursor.
const ScrollOffsetMargin = 3
