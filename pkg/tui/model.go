package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/pkg/tui/containers"
	"github.com/Br0ce/containerctl/pkg/tui/header"
)

type Model struct {
	cli           Client
	err           error
	width, height int
	header        header.Model
	help          help.Model
	containers    containers.Model
}

func newModel(cli Client) Model {
	h := help.New()
	h.ShowAll = true
	return Model{
		cli:        cli,
		header:     header.New(cli.DaemonHost(), cli.DaemonVersion()),
		help:       h,
		containers: containers.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return fetchContainers(m.cli)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.layout(), nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, quitKey):
			return m, tea.Quit
		case key.Matches(msg, helpKey):
			m.help.ShowAll = !m.help.ShowAll
			return m.layout(), nil
		}

	case tickMsg:
		return m, fetchContainers(m.cli)

	case containersMsg:
		m.err = msg.err
		if msg.err == nil {
			m.containers = m.containers.SetShorts(msg.shorts)
			m.header = m.header.SetShorts(msg.shorts)
		}
		// Schedule the next refresh only once this one is done, so fetches
		// never overlap on slow connections.
		return m, tick()
	}

	var cmd tea.Cmd
	m.containers, cmd = m.containers.Update(msg)
	return m, cmd
}

// layout gives the containers pane the space left over by the header and the
// footer, whose height depends on whether the full help is shown.
func (m Model) layout() Model {
	m.header = m.header.SetWidth(m.width)
	m.help.SetWidth(m.width)
	footerH := lipgloss.Height(m.footer())
	m.containers = m.containers.SetSize(m.width, max(m.height-header.Height-footerH, 0))
	return m
}

func (m Model) footer() string {
	return m.help.View(keyMap{table: m.containers.KeyMap()})
}

func (m Model) View() tea.View {
	content := m.containers.View()
	if m.err != nil {
		content = "Error: " + m.err.Error()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, m.header.View(), content, m.footer()))
	v.AltScreen = true
	return v
}
