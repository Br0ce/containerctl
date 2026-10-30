package client

import (
	"cmp"
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	mcont "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mcli "github.com/moby/moby/client"

	"github.com/Br0ce/containerctl/container"
)

// maxHealthChecks bounds the healthcheck results kept per container.
const maxHealthChecks = 5

// Details returns the inspect information of container id.
func (cli *Client) Details(ctx context.Context, id string) (container.Details, error) {
	res, err := cli.client.ContainerInspect(ctx, id, mcli.ContainerInspectOptions{})
	if err != nil {
		return container.Details{}, fmt.Errorf("inspect container %s: %w", id, err)
	}
	return details(res.Container), nil
}

func details(in mcont.InspectResponse) container.Details {
	d := container.Details{
		ID:           in.ID,
		Name:         strings.TrimPrefix(in.Name, "/"),
		ImageID:      in.Image,
		Created:      parseTime(in.Created),
		RestartCount: in.RestartCount,
		Mounts:       mounts(in.Mounts),
	}
	if s := in.State; s != nil {
		d.State = string(s.Status)
		d.Pid = s.Pid
		d.ExitCode = s.ExitCode
		d.OOMKilled = s.OOMKilled
		d.Error = s.Error
		d.StartedAt = parseTime(s.StartedAt)
		d.FinishedAt = parseTime(s.FinishedAt)
		d.Health = healthDetails(s.Health)
	}
	var exposed network.PortSet
	if c := in.Config; c != nil {
		d.Image = c.Image
		d.Entrypoint = c.Entrypoint
		d.Cmd = c.Cmd
		d.WorkingDir = c.WorkingDir
		d.User = c.User
		d.Env = c.Env
		d.Labels = c.Labels
		exposed = c.ExposedPorts
	}
	if h := in.HostConfig; h != nil {
		d.RestartPolicy = restartPolicy(h.RestartPolicy)
		d.Resources = resources(h.Resources)
	}
	var bindings network.PortMap
	if n := in.NetworkSettings; n != nil {
		d.Networks = networks(n.Networks)
		bindings = n.Ports
	}
	d.Ports = ports(exposed, bindings)
	return d
}

// parseTime returns the zero time for timestamps that are unset, which Docker
// reports as "0001-01-01T00:00:00Z".
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.Year() <= 1 {
		return time.Time{}
	}
	return t
}

func healthDetails(h *mcont.Health) *container.Health {
	if h == nil || h.Status == "" || h.Status == mcont.NoHealthcheck {
		return nil
	}
	out := &container.Health{Status: string(h.Status), FailingStreak: h.FailingStreak}
	for _, c := range h.Log[max(len(h.Log)-maxHealthChecks, 0):] {
		if c == nil {
			continue
		}
		out.Checks = append(out.Checks, container.HealthCheck{
			Start:    c.Start,
			ExitCode: c.ExitCode,
			Output:   c.Output,
		})
	}
	return out
}

func restartPolicy(p mcont.RestartPolicy) string {
	if p.Name == "" {
		return string(mcont.RestartPolicyDisabled)
	}
	if p.Name == mcont.RestartPolicyOnFailure && p.MaximumRetryCount > 0 {
		return fmt.Sprintf("%s:%d", p.Name, p.MaximumRetryCount)
	}
	return string(p.Name)
}

func resources(r mcont.Resources) container.Resources {
	out := container.Resources{
		Memory:    r.Memory,
		NanoCPUs:  r.NanoCPUs,
		CPUShares: r.CPUShares,
	}
	if r.PidsLimit != nil && *r.PidsLimit > 0 {
		out.PidsLimit = *r.PidsLimit
	}
	for _, u := range r.Ulimits {
		if u != nil {
			out.Ulimits = append(out.Ulimits, container.Ulimit{Name: u.Name, Soft: u.Soft, Hard: u.Hard})
		}
	}
	return out
}

// ports merges the exposed ports with their host bindings, sorted by
// container port and without the duplicates Docker reports for IPv4 and
// IPv6. A stopped container has no bindings, so its exposed ports are listed
// as unpublished.
func ports(exposed network.PortSet, bindings network.PortMap) []container.Port {
	var out []container.Port
	for p := range exposed {
		if _, ok := bindings[p]; !ok {
			out = append(out, container.Port{Private: p.Num(), Proto: string(p.Proto())})
		}
	}
	for p, bs := range bindings {
		priv := container.Port{Private: p.Num(), Proto: string(p.Proto())}
		if len(bs) == 0 {
			out = append(out, priv)
		}
		for _, b := range bs {
			pub, err := strconv.ParseUint(b.HostPort, 10, 16)
			if err != nil {
				pub = 0
			}
			out = append(out, container.Port{Public: uint16(pub), Private: priv.Private, Proto: priv.Proto})
		}
	}
	slices.SortFunc(out, func(a, b container.Port) int {
		return cmp.Or(
			cmp.Compare(a.Private, b.Private),
			strings.Compare(a.Proto, b.Proto),
			cmp.Compare(a.Public, b.Public),
		)
	})
	return slices.Compact(out)
}

func networks(in map[string]*network.EndpointSettings) []container.Network {
	var out []container.Network
	for name, e := range in {
		if e == nil {
			continue
		}
		n := container.Network{
			Name:    name,
			Gateway: addr(e.Gateway),
			MAC:     e.MacAddress.String(),
			Aliases: e.Aliases,
		}
		if ip := addr(e.IPAddress); ip != "" {
			n.IP = fmt.Sprintf("%s/%d", ip, e.IPPrefixLen)
		}
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b container.Network) int { return strings.Compare(a.Name, b.Name) })
	return out
}

func addr(a netip.Addr) string {
	if !a.IsValid() {
		return ""
	}
	return a.String()
}

// mounts lists volumes by name rather than by their storage location on the
// host.
func mounts(in []mcont.MountPoint) []container.Mount {
	out := make([]container.Mount, 0, len(in))
	for _, m := range in {
		src := m.Source
		if m.Name != "" {
			src = m.Name
		}
		out = append(out, container.Mount{Type: string(m.Type), Source: src, Destination: m.Destination, RW: m.RW})
	}
	slices.SortFunc(out, func(a, b container.Mount) int { return strings.Compare(a.Destination, b.Destination) })
	return out
}
