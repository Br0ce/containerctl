package client

import (
	"net/netip"
	"slices"
	"testing"
	"time"

	mcont "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"

	"github.com/Br0ce/containerctl/container"
)

func TestDetails(t *testing.T) {
	pids := int64(100)
	var log []*mcont.HealthcheckResult
	for i := range 7 {
		log = append(log, &mcont.HealthcheckResult{ExitCode: i})
	}
	in := mcont.InspectResponse{
		ID:      "aaaaaaaaaaaaaaaa",
		Name:    "/web",
		Image:   "sha256:abc",
		Created: "2026-10-30T12:00:00.123456789Z",
		State: &mcont.State{
			Status:     mcont.StateRunning,
			Pid:        42,
			StartedAt:  "2026-10-30T12:00:01Z",
			FinishedAt: "0001-01-01T00:00:00Z",
			Health:     &mcont.Health{Status: mcont.Unhealthy, FailingStreak: 2, Log: log},
		},
		Config: &mcont.Config{
			Image:  "nginx",
			Labels: map[string]string{"a": "b"},
		},
		HostConfig: &mcont.HostConfig{
			RestartPolicy: mcont.RestartPolicy{Name: mcont.RestartPolicyOnFailure, MaximumRetryCount: 3},
			Resources:     mcont.Resources{Memory: 1 << 30, PidsLimit: &pids},
		},
		NetworkSettings: &mcont.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"b": {IPAddress: netip.MustParseAddr("172.18.0.2"), IPPrefixLen: 16},
				"a": {},
			},
		},
		Mounts: []mcont.MountPoint{
			{Type: "volume", Name: "data", Source: "/var/lib/docker/volumes/data/_data", Destination: "/data", RW: true},
			{Type: "bind", Source: "/srv/conf", Destination: "/conf"},
		},
	}

	d := details(in)
	if d.Name != "web" || d.Image != "nginx" || d.ImageID != "sha256:abc" || d.State != "running" || d.Pid != 42 {
		t.Errorf("unexpected general fields: %+v", d)
	}
	if want := time.Date(2026, 10, 30, 12, 0, 0, 123456789, time.UTC); !d.Created.Equal(want) {
		t.Errorf("Created = %v, want %v", d.Created, want)
	}
	if !d.FinishedAt.IsZero() {
		t.Errorf("FinishedAt = %v, want zero", d.FinishedAt)
	}
	if d.RestartPolicy != "on-failure:3" {
		t.Errorf("RestartPolicy = %q", d.RestartPolicy)
	}
	if d.Resources.Memory != 1<<30 || d.Resources.PidsLimit != 100 {
		t.Errorf("Resources = %+v", d.Resources)
	}
	if d.Health == nil || d.Health.Status != "unhealthy" || len(d.Health.Checks) != maxHealthChecks || d.Health.Checks[0].ExitCode != 2 {
		t.Errorf("Health = %+v, want the last %d checks", d.Health, maxHealthChecks)
	}
	if want := []container.Network{{Name: "a"}, {Name: "b", IP: "172.18.0.2/16"}}; !slices.EqualFunc(d.Networks, want, func(a, b container.Network) bool {
		return a.Name == b.Name && a.IP == b.IP && a.Gateway == b.Gateway && a.MAC == b.MAC
	}) {
		t.Errorf("Networks = %+v, want %+v", d.Networks, want)
	}
	want := []container.Mount{
		{Type: "bind", Source: "/srv/conf", Destination: "/conf"},
		{Type: "volume", Source: "data", Destination: "/data", RW: true},
	}
	if !slices.Equal(d.Mounts, want) {
		t.Errorf("Mounts = %+v, want %+v", d.Mounts, want)
	}
}

func TestDetailsWithoutHealthcheck(t *testing.T) {
	d := details(mcont.InspectResponse{
		State:      &mcont.State{Health: &mcont.Health{Status: mcont.NoHealthcheck}},
		HostConfig: &mcont.HostConfig{},
	})
	if d.Health != nil {
		t.Errorf("Health = %+v, want nil", d.Health)
	}
	if d.RestartPolicy != "no" {
		t.Errorf("RestartPolicy = %q, want no", d.RestartPolicy)
	}
}

func TestPorts(t *testing.T) {
	p80 := network.MustParsePort("80/tcp")
	p443 := network.MustParsePort("443/tcp")
	p53 := network.MustParsePort("53/udp")
	exposed := network.PortSet{p80: {}, p443: {}, p53: {}}
	bindings := network.PortMap{
		p80: {
			{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: "8080"},
			{HostIP: netip.MustParseAddr("::"), HostPort: "8080"},
		},
		p443: nil,
	}
	want := []container.Port{
		{Private: 53, Proto: "udp"},
		{Public: 8080, Private: 80, Proto: "tcp"},
		{Private: 443, Proto: "tcp"},
	}
	if got := ports(exposed, bindings); !slices.Equal(got, want) {
		t.Errorf("ports = %+v, want %+v", got, want)
	}
	if got := ports(nil, nil); got != nil {
		t.Errorf("ports(nil, nil) = %+v, want nil", got)
	}
}
