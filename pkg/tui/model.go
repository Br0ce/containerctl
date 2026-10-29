package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/pkg/tui/containers"
)

var quitKey = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))

type Model struct {
	cli        Client
	err        error
	containers containers.Model
}

func newModel(cli Client) Model {
	return Model{cli: cli, containers: containers.New()}
}

func (m Model) Init() tea.Cmd {
	return fetchContainers(m.cli)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.containers = m.containers.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, quitKey) {
			return m, tea.Quit
		}

	case containersMsg:
		m.err = msg.err
		if msg.err == nil {
			m.containers = m.containers.SetShorts(msg.shorts)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.containers, cmd = m.containers.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	content := m.containers.View()
	if m.err != nil {
		content = "Error: " + m.err.Error()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
