package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/registry"
)

type registryFetchResultMsg struct {
	Registry *registry.Registry
	Err      error
}

func (m Model) enterBrowse() (tea.Model, tea.Cmd) {
	m.screen = ScreenSearch
	m.searchLoading = true
	m.searchCategory = -1
	m.searchResults = nil
	m.searchErr = nil
	m.searchQuery = ""
	m.searchQuerySnapshot = ""
	m.searchScroll.reset()
	m.searchInput.Reset()
	m.searching = false

	cmd := m.fetchRegistryCmd()
	return m, tea.Batch(cmd, m.checkSpinner.Tick)
}

func (m Model) handleKeyMsgSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchLoading {
		return m, nil
	}

	if m.searchInput.Focused() {
		switch {
		case key.Matches(msg, SharedKeys.Back): // Esc → revert query, blur
			m.searchInput.SetValue(m.searchQuerySnapshot)
			m.searchQuery = m.searchQuerySnapshot
			m.searchInput.Blur()
			m.applySearchFilter()
			m.searching = false
			return m, nil
		case msg.Type == tea.KeyEnter: // Enter → accept query, blur
			m.searchInput.Blur()
			m.searching = false
			return m, nil
		default: // All other keys → text input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchQuery != m.searchInput.Value() {
				m.searchQuery = m.searchInput.Value()
				m.applySearchFilter()
			}
			return m, cmd
		}
	}
	switch {
	case key.Matches(msg, SharedKeys.Back):
		m.screen = ScreenList
		return m, nil
	case key.Matches(msg, SharedKeys.Quit):
		return m, tea.Quit
	case msg.Type == tea.KeyTab:
		m.cycleCategory()
		m.applySearchFilter()
		return m, nil
	case key.Matches(msg, ListKeys.Up):
		m.searchScroll.moveUp()
		return m, nil
	case key.Matches(msg, ListKeys.Down):
		m.searchScroll.moveDown(len(m.searchResults), m.searchViewHeight())
		return m, nil
	case key.Matches(msg, ListKeys.Install):
		return m.installFromSearch()
	case key.Matches(msg, ListKeys.Search):
		m.searchQuerySnapshot = m.searchQuery
		m.searchInput.Focus()
		m.searching = true
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) handleRegistryFetch(msg registryFetchResultMsg) (tea.Model, tea.Cmd) {
	m.searchLoading = false
	if msg.Err != nil {
		m.searchErr = msg.Err
		return m, nil
	}
	m.searchRegistry = msg.Registry
	m.applySearchFilter()
	return m, nil
}

func (m *Model) cycleCategory() {
	if m.searchRegistry == nil {
		return
	}
	m.searchCategory++
	if m.searchCategory >= len(m.searchRegistry.Categories) {
		m.searchCategory = -1
	}
	m.searchScroll.reset()
}

func (m *Model) applySearchFilter() {
	if m.searchRegistry == nil {
		m.searchResults = nil
		return
	}

	source := m.searchRegistry
	if m.searchCategory >= 0 && m.searchCategory < len(m.searchRegistry.Categories) {
		cat := m.searchRegistry.Categories[m.searchCategory]
		filtered := registry.FilterByCategory(source, cat)
		source = &registry.Registry{Plugins: filtered, Categories: source.Categories}
	}

	m.searchResults = registry.Search(source, m.searchQuery)
	m.searchScroll.reset()
}

func (m *Model) fetchRegistryCmd() tea.Cmd {
	statePath := m.cfg.StatePath
	return func() tea.Msg {
		ctx := context.Background()
		reg, err := registry.Fetch(
			ctx,
			registry.DefaultRegistryURL,
			statePath,
			registry.DefaultCacheTTL,
		)
		return registryFetchResultMsg{Registry: reg, Err: err}
	}
}

func (m Model) installFromSearch() (tea.Model, tea.Cmd) {
	if m.searchScroll.cursor < 0 || m.searchScroll.cursor >= len(m.searchResults) {
		return m, nil
	}

	selected := m.searchResults[m.searchScroll.cursor]

	for _, p := range m.plugins {
		if p.Spec == selected.Repo || p.Name == pluginNameFromRepo(selected.Repo) {
			return m, nil
		}
	}

	if m.cfg.TmuxConf != "" {
		_ = config.AppendPlugin(m.cfg.TmuxConf, selected.Repo)
	}

	name := pluginNameFromRepo(selected.Repo)
	m.plugins = append(m.plugins, PluginItem{
		Name:   name,
		Spec:   selected.Repo,
		Status: StatusNotInstalled,
	})

	m.screen = ScreenProgress
	m.operation = OpInstall
	m.pendingItems = []pendingOp{{
		Name: name,
		Spec: selected.Repo,
		Path: plug.PluginPath(name, m.cfg.PluginPath),
	}}
	m.totalItems = 1
	m.completedItems = 0
	m.results = nil
	m.processing = true
	m.inFlight = 0
	m.inFlightNames = nil
	m.resultScroll.reset()

	cmd := m.dispatchNext()
	return m, cmd
}
