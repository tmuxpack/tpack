package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/tmuxpack/tpack/internal/config"
	"github.com/tmuxpack/tpack/internal/plug"
	"github.com/tmuxpack/tpack/internal/registry"
)

type registryFetchResultMsg struct {
	Registry *registry.Registry
	Err      error
}

func pluginFromRegistryItem(item registry.RegistryItem) (plug.Plugin, error) {
	spec := item.Repo
	if item.Host != "" && item.Host != defaultGitHubHost {
		spec = "https://" + item.Host + "/" + item.Repo
	}
	return plug.ParseSpec(spec, nil)
}

func (m Model) enterBrowse() (tea.Model, tea.Cmd) {
	m.screen = ScreenBrowse
	m.browseLoading = true
	m.browseCategory = -2
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

func (m Model) handleKeyMsgBrowse(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		case msg.Code == tea.KeyEnter: // Enter → accept query, blur
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
	case msg.Code == tea.KeyTab:
		if msg.Mod&tea.ModShift != 0 {
			m.cycleCategoryBackward()
		} else {
			m.cycleCategory()
		}
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
	m.browseRegistry = registry.ExcludeCategories(msg.Registry, m.cfg.HiddenCategories)
	m.applyBrowseFilter()
	return m, nil
}

func (m *Model) cycleCategory() {
	if m.browseRegistry == nil {
		return
	}
	m.browseCategory++
	if m.browseCategory >= len(m.browseRegistry.Categories) {
		m.browseCategory = -2
	}
	m.browseScroll.reset()
}

func (m *Model) cycleCategoryBackward() {
	if m.browseRegistry == nil {
		return
	}
	m.browseCategory--
	if m.browseCategory < -2 {
		m.browseCategory = len(m.browseRegistry.Categories) - 1
	}
	m.browseScroll.reset()
}

func (m *Model) applyBrowseFilter() {
	if m.browseRegistry == nil {
		m.browseResults = nil
		return
	}

	if m.browseCategory == -2 {
		m.browseResults = registry.Newest(m.browseRegistry, 20)
		if m.browseQuery != "" {
			m.browseResults = registry.Search(
				&registry.Registry{Plugins: m.browseResults, Categories: m.browseRegistry.Categories},
				m.browseQuery,
			)
		}
		m.browseScroll.reset()
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
	candidate, err := pluginFromRegistryItem(selected)
	if err != nil {
		m.browseStatus = "Failed to install: " + err.Error()
		return m, nil
	}

	for _, p := range m.plugins {
		if p.Identity == candidate.Identity {
			return m, nil
		}
	}

	path, err := m.cfg.PluginPath.Child(candidate.DirName)
	if err != nil {
		m.browseStatus = "Failed to install: " + err.Error()
		return m, nil
	}

	if m.cfg.TmuxConf != "" {
		_ = config.AppendPlugin(m.cfg.TmuxConf, candidate.Spec)
	}
	m.plugins = append(m.plugins, PluginItem{
		Raw:      candidate.Raw,
		Name:     candidate.Name,
		Identity: candidate.Identity,
		DirName:  candidate.DirName,
		Spec:     candidate.Spec,
		Branch:   candidate.Branch,
		Status:   StatusNotInstalled,
	})

	ops := []pendingOp{{
		Raw:     candidate.Raw,
		Name:    candidate.Name,
		DirName: candidate.DirName,
		Spec:    candidate.Spec,
		Branch:  candidate.Branch,
		Path:    path,
	}}
	cmd := m.initProgress(OpInstall, ops)
	return m, cmd
}
