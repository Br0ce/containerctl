package inspect

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/tui/theme"
)

// keyWidth fits the longest field name, "Restart policy", plus a space.
const keyWidth = 16

// mask hides an environment value. It has a fixed length so that it doesn't
// give away the length of the value.
const mask = "••••••"

// KeyMap extends the viewport keys with jumps to the start and end and the
// environment toggle.
type KeyMap struct {
	viewport.KeyMap
	Top    key.Binding
	Bottom key.Binding
	Env    key.Binding
}

// Model is the inspect pane. It holds no client and performs no I/O; details
// are set by the caller as they arrive.
type Model struct {
	viewport viewport.Model
	keys     KeyMap
	details  *container.Details
	err      error
	// showEnv reveals the environment values, which often hold secrets.
	showEnv bool
	// header styles the section titles like the column titles of the tables.
	header lipgloss.Style
	width  int
}

func New() Model {
	keys := keyMap()
	vp := viewport.New()
	vp.KeyMap = keys.KeyMap
	return Model{viewport: vp, keys: keys, header: theme.Header}
}

// SetDark adapts the section titles to a dark or light terminal background.
func (m Model) SetDark(dark bool) Model {
	m.header = theme.HeaderFor(dark)
	return m.render()
}

// keyMap limits paging and horizontal scrolling to keys that don't collide
// with container actions, like the logs pane does.
func keyMap() KeyMap {
	km := viewport.DefaultKeyMap()
	km.PageUp = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down"))
	km.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down"))
	km.Left = key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "scroll left"))
	km.Right = key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "scroll right"))
	return KeyMap{
		KeyMap: km,
		Top:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/home", "go to start")),
		Bottom: key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G/end", "go to end")),
		Env:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "show env")),
	}
}

// SetSize sets the outer size of the pane, including its border.
func (m Model) SetSize(w, h int) Model {
	m.viewport.SetWidth(max(w-theme.BoxFrameWidth, 0))
	m.viewport.SetHeight(max(h-theme.BoxFrameHeight, 0))
	m.width = w
	return m
}

// Start clears the pane for a new container and hides the environment again.
func (m Model) Start() Model {
	m.details = nil
	m.err = nil
	m.showEnv = false
	m.viewport.SetXOffset(0)
	m.viewport.SetYOffset(0)
	return m.render()
}

// SetDetails shows d and clears any error. The scroll position is kept, so a
// refresh doesn't lose the user's place.
func (m Model) SetDetails(d container.Details) Model {
	m.details = &d
	m.err = nil
	return m.render()
}

// SetErr shows err above the details, which are kept if a refresh fails.
func (m Model) SetErr(err error) Model {
	m.err = err
	return m.render()
}

// KeyMap returns the pane's key bindings for the help footer. The help of the
// environment toggle follows its state.
func (m Model) KeyMap() KeyMap {
	km := m.keys
	if m.showEnv {
		km.Env.SetHelp("e", "hide env")
	}
	return km
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Top):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.Bottom):
			m.viewport.GotoBottom()
			return m, nil
		case key.Matches(msg, m.keys.Env):
			m.showEnv = !m.showEnv
			return m.render(), nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return theme.Box("Inspect", m.width, m.viewport.View())
}

func (m Model) render() Model {
	var lines []string
	if m.err != nil {
		lines = append(lines, " "+theme.Alert.Render("Error: "+m.err.Error()), "")
	}
	switch {
	case m.details != nil:
		lines = append(lines, render(*m.details, m.showEnv, m.header, time.Now())...)
	case m.err == nil:
		lines = append(lines, " "+theme.Dim("Loading…"))
	}
	m.viewport.SetContentLines(lines)
	return m
}

// render lays out the sections of d with titles in the header style. Fields
// that aren't set are left out.
func render(d container.Details, showEnv bool, header lipgloss.Style, now time.Time) []string {
	r := renderer{header: header}
	r.general(d, now)
	r.process(d)
	r.health(d.Health)
	r.network(d)
	r.mounts(d.Mounts)
	r.resources(d.Resources)
	r.env(d.Env, showEnv)
	r.labels(d.Labels)
	return r.lines
}

type renderer struct {
	header lipgloss.Style
	lines  []string
}

func (r *renderer) section(title string) {
	if len(r.lines) > 0 {
		r.lines = append(r.lines, "")
	}
	// The header style pads the title by one column, like the cell padding
	// of the column titles.
	r.lines = append(r.lines, r.header.Render(strings.ToUpper(title)))
}

// field adds a line with key in the key column. It is left out if value is
// empty. A key too long for the column, like a network name, pushes the value
// to the right instead of wrapping.
func (r *renderer) field(key, value string) {
	if value == "" {
		return
	}
	pad := strings.Repeat(" ", max(keyWidth-ansi.StringWidth(key), 1))
	r.lines = append(r.lines, "  "+theme.Key.Render(key)+pad+value)
}

// text adds a line indented to the value column.
func (r *renderer) text(s string) {
	r.lines = append(r.lines, "  "+strings.Repeat(" ", keyWidth)+s)
}

// item adds a line indented below the section title, for lists without a
// key column.
func (r *renderer) item(s string) {
	r.lines = append(r.lines, "  "+s)
}

func (r *renderer) general(d container.Details, now time.Time) {
	r.section("General")
	r.field("ID", d.ID)
	r.field("Name", d.Name)
	r.field("Image", d.Image)
	r.field("Image ID", d.ImageID)
	r.field("Created", timestamp(d.Created, now))
	if d.State != "" {
		r.field("State", theme.StateCell(d.State)+" "+d.State)
	}
	r.field("Started", timestamp(d.StartedAt, now))
	if d.State != "running" && d.State != "paused" {
		r.field("Finished", timestamp(d.FinishedAt, now))
		if !d.FinishedAt.IsZero() {
			r.field("Exit code", strconv.Itoa(d.ExitCode))
		}
	}
	if d.OOMKilled {
		r.field("OOM killed", theme.Alert.Render("yes"))
	}
	if d.Error != "" {
		r.field("Error", theme.Alert.Render(d.Error))
	}
	if d.Pid > 0 {
		r.field("PID", strconv.Itoa(d.Pid))
	}
	r.field("Restart policy", d.RestartPolicy)
	r.field("Restarts", strconv.Itoa(d.RestartCount))
}

func (r *renderer) process(d container.Details) {
	r.section("Process")
	r.field("Entrypoint", command(d.Entrypoint))
	r.field("Command", command(d.Cmd))
	r.field("Working dir", d.WorkingDir)
	r.field("User", d.User)
}

func (r *renderer) health(h *container.Health) {
	if h == nil {
		return
	}
	r.section("Health")
	r.field("Status", theme.HealthMark(h.Status)+" "+h.Status)
	r.field("Failing streak", strconv.Itoa(h.FailingStreak))
	for i, c := range slices.Backward(h.Checks) {
		key := ""
		if i == len(h.Checks)-1 {
			key = "Last checks"
		}
		exit := fmt.Sprintf("exit %d", c.ExitCode)
		if c.ExitCode != 0 {
			exit = theme.Alert.Render(exit)
		}
		out := strings.Split(strings.TrimSpace(c.Output), "\n")
		r.field(key, c.Start.Local().Format(time.TimeOnly)+"  "+exit+"  "+strings.TrimSpace(out[0]))
		for _, l := range out[1:] {
			r.text(theme.Dim(strings.TrimSpace(l)))
		}
	}
}

func (r *renderer) network(d container.Details) {
	if len(d.Networks) == 0 && len(d.Ports) == 0 {
		return
	}
	r.section("Network")
	for _, n := range d.Networks {
		var parts []string
		for _, p := range [][2]string{{"IP", n.IP}, {"Gateway", n.Gateway}, {"MAC", n.MAC}} {
			if p[1] != "" {
				parts = append(parts, theme.Dim(p[0])+" "+p[1])
			}
		}
		if len(parts) == 0 {
			parts = append(parts, theme.Dim("no address"))
		}
		r.field(n.Name, strings.Join(parts, "  "))
		if len(n.Aliases) > 0 {
			r.text(theme.Dim("Aliases") + " " + strings.Join(n.Aliases, ", "))
		}
	}
	for i, p := range d.Ports {
		key := ""
		if i == 0 {
			key = "Ports"
		}
		port := fmt.Sprintf("%d/%s", p.Private, p.Proto)
		if p.Public == 0 {
			r.field(key, port+" "+theme.Dim("(not published)"))
		} else {
			r.field(key, fmt.Sprintf("%d → %s", p.Public, port))
		}
	}
}

func (r *renderer) mounts(ms []container.Mount) {
	if len(ms) == 0 {
		return
	}
	r.section("Mounts")
	for _, m := range ms {
		mode := "ro"
		if m.RW {
			mode = "rw"
		}
		src := m.Source
		if src == "" {
			src = theme.Dim("(none)")
		}
		r.field(m.Type, src+" → "+m.Destination+" "+theme.Dim("("+mode+")"))
	}
}

func (r *renderer) resources(res container.Resources) {
	r.section("Resources")
	if res.Memory == 0 && res.NanoCPUs == 0 && res.CPUShares == 0 && res.PidsLimit == 0 && len(res.Ulimits) == 0 {
		r.item(theme.Dim("No limits set"))
		return
	}
	if res.Memory > 0 {
		r.field("Memory", formatBytes(res.Memory))
	}
	if res.NanoCPUs > 0 {
		r.field("CPUs", strconv.FormatFloat(float64(res.NanoCPUs)/1e9, 'f', -1, 64))
	}
	if res.CPUShares > 0 {
		r.field("CPU shares", strconv.FormatInt(res.CPUShares, 10))
	}
	if res.PidsLimit > 0 {
		r.field("PIDs", strconv.FormatInt(res.PidsLimit, 10))
	}
	for i, u := range res.Ulimits {
		key := ""
		if i == 0 {
			key = "Ulimits"
		}
		r.field(key, fmt.Sprintf("%s %d:%d", u.Name, u.Soft, u.Hard))
	}
}

func (r *renderer) env(env []string, show bool) {
	if len(env) == 0 {
		return
	}
	r.section("Environment")
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		switch {
		case !ok:
			r.item(k)
		case show || v == "":
			r.item(k + "=" + v)
		default:
			r.item(k + "=" + theme.Dim(mask))
		}
	}
}

func (r *renderer) labels(labels map[string]string) {
	if len(labels) == 0 {
		return
	}
	r.section("Labels")
	for _, k := range slices.Sorted(maps.Keys(labels)) {
		r.item(k + "=" + labels[k])
	}
}

// command joins args like a shell command line, quoting those that contain
// whitespace or are empty.
func command(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'") {
			a = strconv.Quote(a)
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

// timestamp renders t in local time with its age, e.g.
// "2026-10-30 14:03:12 (3h ago)", or "" if t is unset.
func timestamp(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(time.DateTime) + " " + theme.Dim("("+age(now.Sub(t))+" ago)")
}

func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", max(int(d.Seconds()), 0))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// formatBytes uses binary units with up to one decimal, e.g. "1.5 GiB".
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	v, i := float64(b)/unit, 0
	for v >= unit && i < 3 {
		v /= unit
		i++
	}
	s := strconv.FormatFloat(v, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0") + " " + []string{"KiB", "MiB", "GiB", "TiB"}[i]
}
