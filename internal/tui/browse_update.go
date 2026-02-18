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
	m.screen = ScreenBrowse
	m.browseLoading = true
	m.browseCategory = -1
	m.browseResults = nil
	m.browseErr = nil
	m.browseQuery = ""
	m.browseQuerySnapshot = ""
	m.browseScroll.reset()
	m.browseInput.Reset()
	m.searching = false

	cmd := m.fetchRegistryCmd()
	return m, tea.Batch(cmd, m.checkSpinner.Tick)
}

func (m Model) handleKeyMsgBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.browseLoading {
		return m, nil
	}

	if m.browseInput.Focused() {
		switch {
		case key.Matches(msg, SharedKeys.Back): // Esc → revert query, blur
			m.browseInput.SetValue(m.browseQuerySnapshot)
			m.browseQuery = m.browseQuerySnapshot
			m.browseInput.Blur()
			m.applyBrowseFilter()
			m.searching = false
			return m, nil
		case msg.Type == tea.KeyEnter: // Enter → accept query, blur
			m.browseInput.Blur()
			m.searching = false
			return m, nil
		default: // All other keys → text input
			var cmd tea.Cmd
			m.browseInput, cmd = m.browseInput.Update(msg)
			if m.browseQuery != m.browseInput.Value() {
				m.browseQuery = m.browseInput.Value()
				m.applyBrowseFilter()
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
		m.applyBrowseFilter()
		return m, nil
	case key.Matches(msg, ListKeys.Up):
		m.browseScroll.moveUp()
		return m, nil
	case key.Matches(msg, ListKeys.Down):
		m.browseScroll.moveDown(len(m.browseResults), m.browseViewHeight())
		return m, nil
	case key.Matches(msg, ListKeys.Install):
		return m.installFromBrowse()
	case key.Matches(msg, BrowseKeys.Open):
		return m.openFromBrowse()
	case key.Matches(msg, ListKeys.Search):
		m.browseQuerySnapshot = m.browseQuery
		m.browseInput.Focus()
		m.searching = true
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) handleRegistryFetch(msg registryFetchResultMsg) (tea.Model, tea.Cmd) {
	m.browseLoading = false
	if msg.Err != nil {
		m.browseErr = msg.Err
		return m, nil
	}
	m.browseRegistry = msg.Registry
	m.applyBrowseFilter()
	return m, nil
}

func (m *Model) cycleCategory() {
	if m.browseRegistry == nil {
		return
	}
	m.browseCategory++
	if m.browseCategory >= len(m.browseRegistry.Categories) {
		m.browseCategory = -1
	}
	m.browseScroll.reset()
}

func (m *Model) applyBrowseFilter() {
	if m.browseRegistry == nil {
		m.browseResults = nil
		return
	}

	source := m.browseRegistry
	if m.browseCategory >= 0 && m.browseCategory < len(m.browseRegistry.Categories) {
		cat := m.browseRegistry.Categories[m.browseCategory]
		filtered := registry.FilterByCategory(source, cat)
		source = &registry.Registry{Plugins: filtered, Categories: source.Categories}
	}

	m.browseResults = registry.Search(source, m.browseQuery)
	m.browseScroll.reset()
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

func (m Model) installFromBrowse() (tea.Model, tea.Cmd) {
	if m.browseScroll.cursor < 0 || m.browseScroll.cursor >= len(m.browseResults) {
		return m, nil
	}

	selected := m.browseResults[m.browseScroll.cursor]

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
