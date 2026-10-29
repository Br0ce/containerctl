package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Br0ce/containerctl/pkg/container"
)

type fakeClient struct {
	shorts []container.Short
	err    error
}

func (f *fakeClient) AllShorts(ctx context.Context) ([]container.Short, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("expected a deadline on the request context")
	}
	return f.shorts, f.err
}

var shorts = []container.Short{
	{ID: "aaaaaaaaaaaaaaaa", Name: "web", Image: "nginx", Status: "Up", State: "running"},
}

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func TestInitLoadsContainers(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 10})

	m, _ = update(m, m.Init()())
	if m.err != nil {
		t.Fatalf("unexpected error: %v", m.err)
	}
	if s, ok := m.containers.Selected(); !ok || s.ID != shorts[0].ID {
		t.Errorf("selected = %+v, want %s", s, shorts[0].ID)
	}
}

func TestFetchError(t *testing.T) {
	m := newModel(&fakeClient{err: errors.New("boom")})
	m, _ = update(m, m.Init()())
	if !strings.Contains(m.View().Content, "boom") {
		t.Errorf("view does not show the error: %q", m.View().Content)
	}
}

func TestQuit(t *testing.T) {
	m := newModel(&fakeClient{})
	_, cmd := update(m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}

func TestContainersMsgSchedulesTick(t *testing.T) {
	m := newModel(&fakeClient{})
	for _, msg := range []containersMsg{{shorts: shorts}, {err: errors.New("boom")}} {
		if _, cmd := update(m, msg); cmd == nil {
			t.Errorf("expected tick command after %+v", msg)
		}
	}
}

func TestTickFetchesContainers(t *testing.T) {
	m := newModel(&fakeClient{shorts: shorts})
	_, cmd := update(m, tickMsg{})
	if cmd == nil {
		t.Fatal("expected fetch command")
	}
	msg, ok := cmd().(containersMsg)
	if !ok {
		t.Fatalf("expected containersMsg, got %T", msg)
	}
	if len(msg.shorts) != len(shorts) {
		t.Errorf("got %d shorts, want %d", len(msg.shorts), len(shorts))
	}
}
