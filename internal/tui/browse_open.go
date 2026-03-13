package tui

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type clearBrowseStatusMsg struct{}

func (m Model) openFromBrowse() (tea.Model, tea.Cmd) {
	if m.browseScroll.cursor < 0 || m.browseScroll.cursor >= len(m.browseResults) {
		return m, nil
	}

	selected := m.browseResults[m.browseScroll.cursor]
	host := selected.Host
	if host == "" {
		host = defaultGitHubHost
	}
	url := "https://" + host + "/" + selected.Repo

	m.browseStatus = "Copied to clipboard: " + url

	return m, tea.Batch(
		tea.SetClipboard(url),
		openURLCmd(url),
		tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearBrowseStatusMsg{}
		}),
	)
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_ = openURL(ctx, url)
		return nil
	}
}

// logic taken from https://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8
func openURL(ctx context.Context, url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd.exe"
		args = []string{"/c", "rundll32", "url.dll,FileProtocolHandler",
			strings.ReplaceAll(url, "&", "^&")}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		if isWSL(ctx) {
			cmd = "cmd.exe"
			args = []string{"start", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}

	return exec.CommandContext(ctx, cmd, args...).Start()
}

func isWSL(ctx context.Context) bool {
	releaseData, err := exec.CommandContext(ctx, "uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}
