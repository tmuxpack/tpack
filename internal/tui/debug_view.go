package tui

import (
	"fmt"
	"strings"
)

// viewDebug renders the debug information screen.
func (m *Model) viewDebug() string {
	var b strings.Builder

	b.WriteString(m.centerText(m.theme.TitleStyle.Render("  Debug Info  ")))
	b.WriteString("\n")

	ver := m.version
	if ver == "" {
		ver = "unknown"
	}
	bin := m.binaryPath
	if bin == "" {
		bin = "unknown"
	}

	b.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.HelpKeyStyle.Render("Version:"), ver))
	b.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.HelpKeyStyle.Render("Binary:"), bin))

	help := m.centerText(m.theme.renderHelp(m.width, "esc", "back", "q", "quit"))

	return padToBottom(b.String(), help, m.height)
}
