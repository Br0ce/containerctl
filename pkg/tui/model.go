package tui

import (
	"context"
	"errors"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/pkg/file"
	"github.com/Br0ce/containerctl/pkg/tui/containers"
	"github.com/Br0ce/containerctl/pkg/tui/files"
	"github.com/Br0ce/containerctl/pkg/tui/header"
	"github.com/Br0ce/containerctl/pkg/tui/logs"
)

type page int

const (
	containersPage page = iota
	logsPage
	filesPage
)

type Model struct {
	cli           Client
	err           error
	width, height int
	page          page
	header        header.Model
	help          help.Model
	containers    containers.Model
	logs          logs.Model
	files         files.Model
	// logSession identifies the current log stream, so that lines still in
	// flight from a previous stream are dropped.
	logSession int
	stopLogs   context.CancelFunc
	// filesSession identifies the current directory request, so that
	// listings still in flight from a previous request are dropped.
	filesSession int
}

func newModel(cli Client) Model {
	h := help.New()
	h.ShowAll = true
	return Model{
		cli:        cli,
		header:     header.New(cli.DaemonHost(), cli.DaemonVersion()),
		help:       h,
		containers: containers.New(),
		logs:       logs.New(),
		files:      files.New(),
		stopLogs:   func() {},
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
			m.stopLogs()
			return m, tea.Quit
		case key.Matches(msg, helpKey):
			m.help.ShowAll = !m.help.ShowAll
			return m.layout(), nil
		case m.page == containersPage && key.Matches(msg, logsKey):
			return m.openLogs()
		case m.page == containersPage && key.Matches(msg, filesKey):
			return m.openFiles()
		case m.page == filesPage && key.Matches(msg, openKey):
			return m.openDir()
		case m.page != containersPage && key.Matches(msg, backKey):
			m.stopLogs()
			m.filesSession++
			m.page = containersPage
			m.header = m.header.ClearContainer()
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

	case logsMsg:
		if msg.session != m.logSession {
			return m, nil
		}
		m.logs = m.logs.Append(msg.lines...)
		if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
			m.logs = m.logs.Append("Error: " + msg.err.Error())
		}
		if msg.done {
			return m, nil
		}
		return m, waitForLogs(msg.session, msg.stream)

	case filesMsg:
		if msg.session != m.filesSession {
			return m, nil
		}
		if msg.err != nil {
			m.files = m.files.SetErr(msg.err)
			return m, nil
		}
		// The client resolves an empty root path to the working directory,
		// which is only known from the listing.
		path := msg.root.Path
		if len(msg.files) > 0 {
			path = msg.files[0].DisplayName
		}
		m.files = m.files.SetFiles(path, msg.files)
		return m, nil
	}

	var cmd tea.Cmd
	switch m.page {
	case containersPage:
		m.containers, cmd = m.containers.Update(msg)
	case logsPage:
		m.logs, cmd = m.logs.Update(msg)
	case filesPage:
		m.files, cmd = m.files.Update(msg)
	}
	return m, cmd
}

// openLogs switches to the logs page and starts streaming the logs of the
// selected container, stopping any previous stream.
func (m Model) openLogs() (Model, tea.Cmd) {
	s, ok := m.containers.Selected()
	if !ok {
		return m, nil
	}
	m.stopLogs()
	m.logSession++
	var cmd tea.Cmd
	cmd, m.stopLogs = streamLogs(m.cli, m.logSession, s.ID)
	m.logs = m.logs.Start()
	m.header = m.header.SetContainer(s)
	m.page = logsPage
	return m.layout(), cmd
}

// openFiles switches to the files page and lists the working directory of the
// selected container.
func (m Model) openFiles() (Model, tea.Cmd) {
	s, ok := m.containers.Selected()
	if !ok {
		return m, nil
	}
	m.filesSession++
	m.files = m.files.Start()
	m.header = m.header.SetContainer(s)
	m.page = filesPage
	return m.layout(), fetchFiles(m.cli, m.filesSession, file.Info{ContainerID: s.ID})
}

// openDir lists the selected directory. The current listing stays until the
// new one arrives.
func (m Model) openDir() (Model, tea.Cmd) {
	f, ok := m.files.Selected()
	if !ok || !f.IsDir {
		return m, nil
	}
	m.filesSession++
	return m, fetchFiles(m.cli, m.filesSession, f)
}

// layout gives the active pane the space left over by the header and the
// footer, whose height depends on the page and on whether the full help is
// shown.
func (m Model) layout() Model {
	m.header = m.header.SetWidth(m.width)
	m.help.SetWidth(m.width)
	bodyH := max(m.height-header.Height-lipgloss.Height(m.footer()), 0)
	m.containers = m.containers.SetSize(m.width, bodyH)
	m.logs = m.logs.SetSize(m.width, bodyH)
	m.files = m.files.SetSize(m.width, bodyH)
	return m
}

func (m Model) footer() string {
	switch m.page {
	case logsPage:
		return m.help.View(logsKeyMap{logs: m.logs.KeyMap()})
	case filesPage:
		return m.help.View(filesKeyMap{table: m.files.KeyMap()})
	}
	return m.help.View(containersKeyMap{table: m.containers.KeyMap()})
}

func (m Model) View() tea.View {
	var content string
	switch {
	case m.page == logsPage:
		content = m.logs.View()
	case m.page == filesPage:
		content = m.files.View()
	case m.err != nil:
		content = "Error: " + m.err.Error()
	default:
		content = m.containers.View()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, m.header.View(), content, m.footer()))
	v.AltScreen = true
	return v
}
