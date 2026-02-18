package tui

import (
	"fmt"
	"strings"
)

const unknownPlaceholder = "unknown"

// viewDebug renders the debug information screen.
func (m *Model) viewDebug() string {
	var b strings.Builder

	b.WriteString(m.centerText(m.theme.TitleStyle.Render("  Debug Info  ")))
	b.WriteString("\n")

	ver := m.version
	if ver == "" {
		ver = unknownPlaceholder
	}
	bin := m.binaryPath
	if bin == "" {
		bin = unknownPlaceholder
	}

	fmt.Fprintf(&b, "  %s  %s\n", m.theme.HelpKeyStyle.Render("Version:"), ver)
	fmt.Fprintf(&b, "  %s  %s\n", m.theme.HelpKeyStyle.Render("Binary:"), bin)

	help := m.centerText(m.theme.renderHelp(m.width, "esc", "back", "q", "quit"))

	return padToBottom(b.String(), help, m.height)
}
