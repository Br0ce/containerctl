package stats

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/Br0ce/containerctl/container"
	"github.com/Br0ce/containerctl/tui/theme"
)

const (
	// keyWidth fits the longest field name, "Received", plus a space, and
	// matches the key column of the inspect pane.
	keyWidth = 16
	barWidth = 20
)

// Model is the stats pane. It holds no client and performs no I/O; samples are
// set by the caller as they arrive. It has no keys of its own.
type Model struct {
	viewport viewport.Model
	// cur is the latest sample and prev the one before, from which the
	// network rates are derived; both are nil until they arrive.
	cur, prev *container.Stats
	err       error
	// header styles the section titles like the column titles of the tables.
	header lipgloss.Style
	width  int
}

func New() Model {
	return Model{viewport: viewport.New(), header: theme.Header}
}

// SetDark adapts the section titles to a dark or light terminal background.
func (m Model) SetDark(dark bool) Model {
	m.header = theme.HeaderFor(dark)
	return m.render()
}

// SetSize sets the outer size of the pane, including its border.
func (m Model) SetSize(w, h int) Model {
	m.viewport.SetWidth(max(w-theme.BoxFrameWidth, 0))
	m.viewport.SetHeight(max(h-theme.BoxFrameHeight, 0))
	m.width = w
	return m.render()
}

// Start clears the pane for a new stats session.
func (m Model) Start() Model {
	m.cur, m.prev = nil, nil
	m.err = nil
	return m.render()
}

// SetStats shows s, the next sample of the stream.
func (m Model) SetStats(s container.Stats) Model {
	m.prev, m.cur = m.cur, &s
	return m.render()
}

// SetErr shows err above the last sample, which is kept.
func (m Model) SetErr(err error) Model {
	m.err = err
	return m.render()
}

func (m Model) View() string {
	return theme.Box("Stats", m.width, m.viewport.View())
}

func (m Model) render() Model {
	var lines []string
	if m.err != nil {
		lines = append(lines, " "+theme.Alert.Render("Error: "+m.err.Error()), "")
	}
	switch {
	case m.cur != nil && m.cur.Read.IsZero():
		lines = append(lines, " "+theme.Dim("Container is not running"))
	case m.cur != nil:
		lines = append(lines, render(*m.cur, m.prev, m.header, barWidth)...)
	case m.err == nil:
		lines = append(lines, " "+theme.Dim("Loading…"))
	}
	m.viewport.SetContentLines(lines)
	return m
}

// render lays out the sections of s with titles in the header style. The
// network rates need the previous sample prev and are left out without it.
func render(s container.Stats, prev *container.Stats, header lipgloss.Style, barW int) []string {
	r := renderer{header: header}

	r.section("CPU")
	r.field("Usage", bar(s.CPUPercent/float64(max(s.OnlineCPUs, 1)*100), barW)+
		fmt.Sprintf("  %5.1f%%  ", s.CPUPercent)+theme.Dim(fmt.Sprintf("of %d CPUs", s.OnlineCPUs)))

	r.section("Memory")
	var memShare float64
	if s.MemLimit > 0 {
		memShare = float64(s.MemUsage) / float64(s.MemLimit)
	}
	r.field("Usage", bar(memShare, barW)+
		fmt.Sprintf("  %5.1f%%  ", memShare*100)+theme.Bytes(s.MemUsage)+" / "+theme.Bytes(s.MemLimit))

	r.section("Network")
	rx, tx := "", ""
	if prev != nil && !prev.Read.IsZero() {
		if dt := s.Read.Sub(prev.Read); dt > 0 {
			rx = "  " + theme.Dim(rate(prev.NetRx, s.NetRx, dt))
			tx = "  " + theme.Dim(rate(prev.NetTx, s.NetTx, dt))
		}
	}
	r.field("Received", theme.Bytes(s.NetRx)+rx)
	r.field("Sent", theme.Bytes(s.NetTx)+tx)
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

func (r *renderer) field(key, value string) {
	r.lines = append(r.lines, "  "+theme.Key.Render(key)+strings.Repeat(" ", keyWidth-len(key))+value)
}

// bar renders share, clamped to [0, 1], as a bar of width cells.
func bar(share float64, width int) string {
	n := int(min(max(share, 0), 1)*float64(width) + 0.5)
	return theme.Key.Render(strings.Repeat("█", n)) + theme.Dim(strings.Repeat("░", width-n))
}

// rate is the change from before to after over dt per second, e.g.
// "1.5 KiB/s". A counter that went down was reset by a restart, so the rate is
// unknown and shown as 0.
func rate(before, after uint64, dt time.Duration) string {
	if after < before {
		return "0 B/s"
	}
	return theme.Bytes(uint64(float64(after-before)/dt.Seconds())) + "/s"
}
