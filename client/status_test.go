package client

import (
	"net/netip"
	"slices"
	"testing"

	mcont "github.com/moby/moby/api/types/container"

	"github.com/Br0ce/containerctl/container"
)

func TestSplitStatus(t *testing.T) {
	tests := []struct {
		status, want, age string
	}{
		{"Up Less than a second", "Up", "<1s"},
		{"Up 1 second", "Up", "1s"},
		{"Up 42 seconds", "Up", "42s"},
		{"Up About a minute (unhealthy)", "Up", "1m"},
		{"Up 5 minutes (health: starting)", "Up", "5m"},
		{"Up About an hour", "Up", "1h"},
		{"Up 2 hours (Paused)", "Up (Paused)", "2h"},
		{"Up 3 days (healthy)", "Up", "3d"},
		{"Up 2 weeks", "Up", "2w"},
		{"Up 4 months", "Up", "4mo"},
		{"Up 2 years", "Up", "2y"},
		{"Exited (0) 3 minutes ago", "Exited (0)", "3m"},
		{"Exited (137) About an hour ago", "Exited (137)", "1h"},
		{"Restarting (1) 5 seconds ago", "Restarting (1)", "5s"},
		{"Created", "Created", ""},
		{"Removal In Progress", "Removal In Progress", ""},
		{"Dead", "Dead", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		got, age := splitStatus(tt.status)
		if got != tt.want || age != tt.age {
			t.Errorf("splitStatus(%q) = %q, %q, want %q, %q", tt.status, got, age, tt.want, tt.age)
		}
	}
}

func TestPublishedPorts(t *testing.T) {
	v4, v6 := netip.MustParseAddr("0.0.0.0"), netip.MustParseAddr("::")
	ports := []mcont.PortSummary{
		{IP: v4, PublicPort: 8888, PrivatePort: 8080, Type: "tcp"},
		{IP: v6, PublicPort: 8888, PrivatePort: 8080, Type: "tcp"},
		{PrivatePort: 9000, Type: "tcp"},
		{IP: v4, PublicPort: 53, PrivatePort: 53, Type: "udp"},
		{IP: v4, PublicPort: 53, PrivatePort: 53, Type: "tcp"},
	}
	want := []container.Port{
		{Public: 53, Private: 53, Proto: "tcp"},
		{Public: 53, Private: 53, Proto: "udp"},
		{Public: 8888, Private: 8080, Proto: "tcp"},
	}
	if got := publishedPorts(ports); !slices.Equal(got, want) {
		t.Errorf("publishedPorts = %v, want %v", got, want)
	}
	if got := publishedPorts(nil); got != nil {
		t.Errorf("publishedPorts(nil) = %v, want nil", got)
	}
}

func TestHealth(t *testing.T) {
	if got := health(nil); got != "" {
		t.Errorf("health(nil) = %q, want empty", got)
	}
	if got := health(&mcont.HealthSummary{Status: mcont.Unhealthy}); got != "unhealthy" {
		t.Errorf("health(unhealthy) = %q, want unhealthy", got)
	}
}
