package tui

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/file"
	"github.com/Br0ce/containerctl/tui/containers"
	"github.com/Br0ce/containerctl/tui/files"
	"github.com/Br0ce/containerctl/tui/header"
	"github.com/Br0ce/containerctl/tui/inspect"
	"github.com/Br0ce/containerctl/tui/logs"
	"github.com/Br0ce/containerctl/tui/stats"
	"github.com/Br0ce/containerctl/tui/theme"
)

type page int

const (
	containersPage page = iota
	logsPage
	filesPage
	inspectPage
	statsPage
)

// footerIndent lines the footer up with the text in the header boxes, which
// sits behind the border and one column of padding.
const footerIndent = 2

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
	inspect       inspect.Model
	stats         stats.Model
	// logSession identifies the current log stream, so that lines still in
	// flight from a previous stream are dropped.
	logSession int
	stopLogs   context.CancelFunc
	// filesSession identifies the current directory request, so that
	// listings still in flight from a previous request are dropped.
	filesSession int
	// inspectSession identifies the current details request, so that
	// details still in flight from a previous request are dropped.
	inspectSession int
	// statsSession identifies the current stats stream, so that samples
	// still in flight from a previous stream are dropped.
	statsSession int
	stopStats    context.CancelFunc
	// inspected is the ID of the container shown on the inspect page.
	inspected string
	// removing is the container awaiting confirmation for removal; its ID
	// is empty if no removal is pending.
	removing container.Short
	// copying is the file awaiting a destination to be copied to; its Path
	// is empty if no copy is pending. dest is the prompt for the
	// destination.
	copying file.Info
	dest    textinput.Model
	// status is a short message shown above the footer on every page, such
	// as a failed container action; it is empty if nothing is shown.
	// statusSession identifies the current message, so that the timer of
	// an earlier one doesn't clear it.
	status        string
	statusStyle   lipgloss.Style
	statusSession int
}

func newModel(cli Client) Model {
	return Model{
		cli:        cli,
		header:     header.New(cli.DaemonHost(), cli.DaemonVersion()),
		help:       help.New(),
		containers: containers.New(),
		logs:       logs.New(),
		files:      files.New(),
		dest:       textinput.New(),
		inspect:    inspect.New(),
		stats:      stats.New(),
		stopLogs:   func() {},
		stopStats:  func() {},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchContainers(m.cli), fetchEngine(m.cli), tea.RequestBackgroundColor)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.layout(), nil

	case tea.KeyPressMsg:
		// An error from the last shell session is shown until the next key.
		m.containers = m.containers.SetErr(nil)
		if m.removing.ID != "" {
			return m.confirmRemove(msg)
		}
		if m.copying.Path != "" {
			return m.promptCopy(msg)
		}
		switch {
		case key.Matches(msg, quitKey):
			m.stopLogs()
			m.stopStats()
			return m, tea.Quit
		case key.Matches(msg, helpKey):
			m.help.ShowAll = !m.help.ShowAll
			return m.layout(), nil
		case m.page == containersPage && key.Matches(msg, logsKey):
			return m.openLogs()
		case m.page == containersPage && key.Matches(msg, filesKey):
			return m.openFiles()
		case m.page == containersPage && key.Matches(msg, inspectKey):
			return m.openInspect()
		case m.page == containersPage && key.Matches(msg, statsKey):
			return m.openStats()
		case m.page == containersPage && key.Matches(msg, termKey):
			if s, ok := m.containers.Selected(); ok {
				return m, openTerminal(m.cli, s.ID)
			}
			return m, nil
		case m.page == containersPage && key.Matches(msg, startKey):
			return m.runAction("start", m.cli.StartContainer)
		case m.page == containersPage && key.Matches(msg, stopKey):
			return m.runAction("stop", m.cli.StopContainer)
		case m.page == containersPage && key.Matches(msg, pauseKey):
			return m.runAction("pause", m.cli.PauseContainer)
		case m.page == containersPage && key.Matches(msg, unpauseKey):
			return m.runAction("unpause", m.cli.UnpauseContainer)
		case m.page == containersPage && key.Matches(msg, restartKey):
			return m.runAction("restart", m.cli.RestartContainer)
		case m.page == containersPage && key.Matches(msg, removeKey):
			if s, ok := m.containers.Selected(); ok {
				m.removing = s
				return m.layout(), nil
			}
			return m, nil
		case m.page == filesPage && key.Matches(msg, openKey):
			return m.openDir()
		case m.page == filesPage && key.Matches(msg, copyKey):
			return m.startCopy()
		case m.page == inspectPage && key.Matches(msg, refreshKey):
			m.inspectSession++
			return m, fetchDetails(m.cli, m.inspectSession, m.inspected)
		case m.page != containersPage && key.Matches(msg, backKey):
			m.stopLogs()
			m.stopStats()
			m.filesSession++
			m.inspectSession++
			m.page = containersPage
			m.header = m.header.ClearContainer()
			return m.layout(), nil
		}

	case tea.BackgroundColorMsg:
		m.containers = m.containers.SetDark(msg.IsDark())
		m.files = m.files.SetDark(msg.IsDark())
		m.inspect = m.inspect.SetDark(msg.IsDark())
		m.stats = m.stats.SetDark(msg.IsDark())
		return m, nil

	case tickMsg:
		return m, fetchContainers(m.cli)

	case engineMsg:
		// Without the host's information the header keeps showing the
		// connection target and client API version; nothing else depends on
		// it.
		if msg.err == nil {
			m.header = m.header.SetEngine(msg.info)
		}
		return m, nil

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

	case statsMsg:
		if msg.session != m.statsSession || msg.done {
			return m, nil
		}
		if msg.err != nil {
			if !errors.Is(msg.err, context.Canceled) {
				m.stats = m.stats.SetErr(msg.err)
			}
			return m, waitForStats(msg.session, msg.stream)
		}
		m.stats = m.stats.SetStats(msg.stats)
		return m, waitForStats(msg.session, msg.stream)

	case terminalMsg:
		m.containers = m.containers.SetErr(msg.err)
		return m, nil

	case actionMsg:
		if msg.err == nil {
			return m, nil
		}
		return m.setStatus(theme.Alert, actionStatus(msg))

	case copyMsg:
		if msg.err != nil {
			return m.setStatus(theme.Alert, actionStatus(actionMsg{verb: "copy", name: msg.name, err: msg.err}))
		}
		return m.setStatus(theme.Value, "copied "+msg.name+" to "+msg.dst)

	case clearStatusMsg:
		if msg.session != m.statusSession {
			return m, nil
		}
		m.status = ""
		return m.layout(), nil

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

	case detailsMsg:
		if msg.session != m.inspectSession {
			return m, nil
		}
		if msg.err != nil {
			m.inspect = m.inspect.SetErr(msg.err)
			return m, nil
		}
		m.inspect = m.inspect.SetDetails(msg.details)
		return m, nil
	}

	var cmd tea.Cmd
	if m.copying.Path != "" {
		// Lets the prompt's cursor blink.
		m.dest, cmd = m.dest.Update(msg)
		return m, cmd
	}
	switch m.page {
	case containersPage:
		m.containers, cmd = m.containers.Update(msg)
	case logsPage:
		m.logs, cmd = m.logs.Update(msg)
	case filesPage:
		m.files, cmd = m.files.Update(msg)
	case inspectPage:
		m.inspect, cmd = m.inspect.Update(msg)
	}
	return m, cmd
}

// runAction applies action to the selected container without waiting for it.
// The status line is cleared, so that it only ever reports on the latest
// action.
func (m Model) runAction(verb string, action func(ctx context.Context, id string) error) (Model, tea.Cmd) {
	s, ok := m.containers.Selected()
	if !ok {
		return m, nil
	}
	m = m.clearStatus()
	return m, containerAction(verb, action, s)
}

// setStatus shows text in style on the status line for statusDuration.
func (m Model) setStatus(style lipgloss.Style, text string) (Model, tea.Cmd) {
	m.status = text
	m.statusStyle = style
	m.statusSession++
	return m.layout(), clearStatus(m.statusSession)
}

// clearStatus hides the status line right away; the timer of the hidden
// message is ignored.
func (m Model) clearStatus() Model {
	if m.status == "" {
		return m
	}
	m.status = ""
	m.statusSession++
	return m.layout()
}

// confirmRemove ends a pending removal: y removes the container without
// waiting for it, any other key cancels. The key is not passed on.
func (m Model) confirmRemove(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	s := m.removing
	m.removing = container.Short{}
	m = m.layout()
	if !key.Matches(msg, confirmKey) {
		return m, nil
	}
	m = m.clearStatus()
	return m, containerAction("remove", m.cli.RemoveContainer, s)
}

// startCopy prompts for the destination of the selected file or directory,
// proposing its name in the working directory. The parent entry is skipped.
func (m Model) startCopy() (Model, tea.Cmd) {
	f, ok := m.files.Selected()
	if !ok || f.Name == ".." {
		return m, nil
	}
	m.copying = f
	m.dest.Reset()
	m.dest.Prompt = "Copy " + f.Name + " to: "
	m.dest.SetValue(f.Name)
	cmd := m.dest.Focus()
	return m.layout(), cmd
}

// promptCopy handles a key while the copy destination is entered: enter
// copies without waiting for it, esc cancels, and any other key edits the
// destination. The key is not passed on.
func (m Model) promptCopy(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, submitKey):
		f, dst := m.copying, strings.TrimSpace(m.dest.Value())
		m = m.endCopy()
		if dst == "" {
			return m, nil
		}
		m = m.clearStatus()
		return m, copyFile(m.cli, f, dst)
	case key.Matches(msg, cancelKey):
		return m.endCopy(), nil
	}
	var cmd tea.Cmd
	m.dest, cmd = m.dest.Update(msg)
	return m, cmd
}

// endCopy closes the prompt for the copy destination.
func (m Model) endCopy() Model {
	m.copying = file.Info{}
	m.dest.Blur()
	return m.layout()
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

// openInspect switches to the inspect page and fetches the details of the
// selected container.
func (m Model) openInspect() (Model, tea.Cmd) {
	s, ok := m.containers.Selected()
	if !ok {
		return m, nil
	}
	m.inspectSession++
	m.inspected = s.ID
	m.inspect = m.inspect.Start()
	m.header = m.header.SetContainer(s)
	m.page = inspectPage
	return m.layout(), fetchDetails(m.cli, m.inspectSession, s.ID)
}

// openStats switches to the stats page and starts streaming the resource
// usage of the selected container, stopping any previous stream.
func (m Model) openStats() (Model, tea.Cmd) {
	s, ok := m.containers.Selected()
	if !ok {
		return m, nil
	}
	m.stopStats()
	m.statsSession++
	var cmd tea.Cmd
	cmd, m.stopStats = streamStats(m.cli, m.statsSession, s.ID)
	m.stats = m.stats.Start()
	m.header = m.header.SetContainer(s)
	m.page = statsPage
	return m.layout(), cmd
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
	m.help.SetWidth(m.footerWidth())
	m.dest.SetWidth(max(m.footerWidth()-lipgloss.Width(m.dest.Prompt)-1, 1))
	bodyH := max(m.height-header.Height-lipgloss.Height(m.footer()), 0)
	m.containers = m.containers.SetSize(m.width, bodyH)
	m.logs = m.logs.SetSize(m.width, bodyH)
	m.files = m.files.SetSize(m.width, bodyH)
	m.inspect = m.inspect.SetSize(m.width, bodyH)
	m.stats = m.stats.SetSize(m.width, bodyH)
	return m
}

// footer renders the help for the active page, the confirmation prompt while
// a removal is pending or the destination prompt while a copy is pending,
// below the status line if a message is shown.
// Each line is indented by footerIndent and cut to the window width, since
// help lets items overflow when no ellipsis fits after the last one that does.
func (m Model) footer() string {
	var lines []string
	if m.status != "" {
		lines = append(lines, m.statusStyle.Render(m.status))
	}
	switch {
	case m.removing.ID != "":
		lines = append(lines, "Remove container "+m.removing.Name+"? (y/N)")
	case m.copying.Path != "":
		lines = append(lines, m.dest.View())
	default:
		lines = append(lines, strings.Split(m.help.View(m.keyMap()), "\n")...)
	}
	indent := strings.Repeat(" ", footerIndent)
	for i, l := range lines {
		lines[i] = indent + ansi.Truncate(l, m.footerWidth(), "…")
	}
	return strings.Join(lines, "\n")
}

// footerWidth is the width left to the footer by its indent.
func (m Model) footerWidth() int {
	return max(m.width-footerIndent, 0)
}

// keyMap returns the keys shown in the help for the active page.
func (m Model) keyMap() help.KeyMap {
	var keys help.KeyMap
	switch m.page {
	case logsPage:
		keys = logsKeyMap{logs: m.logs.KeyMap()}
	case filesPage:
		keys = filesKeyMap{table: m.files.KeyMap()}
	case inspectPage:
		keys = inspectKeyMap{inspect: m.inspect.KeyMap()}
	case statsPage:
		keys = statsKeyMap{}
	default:
		keys = containersKeyMap{table: m.containers.KeyMap()}
	}
	return keys
}

func (m Model) View() tea.View {
	var content string
	switch {
	case m.page == logsPage:
		content = m.logs.View()
	case m.page == filesPage:
		content = m.files.View()
	case m.page == inspectPage:
		content = m.inspect.View()
	case m.page == statsPage:
		content = m.stats.View()
	case m.err != nil:
		content = "Error: " + m.err.Error()
	default:
		content = m.containers.View()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, m.header.View(), content, m.footer()))
	v.AltScreen = true
	return v
}
