package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	mcont "github.com/moby/moby/api/types/container"
	mcli "github.com/moby/moby/client"

	"github.com/Br0ce/containerctl/container"
)

type StatsSeq = iter.Seq2[container.Stats, error]

// Stats streams the resource usage of container id, about one sample per
// second, until ctx is canceled or the consumer stops. A container that isn't
// running yields empty samples.
func (cli *Client) Stats(ctx context.Context, id string) StatsSeq {
	return func(yield func(container.Stats, error) bool) {
		res, err := cli.client.ContainerStats(ctx, id, mcli.ContainerStatsOptions{Stream: true})
		if err != nil {
			yield(container.Stats{}, fmt.Errorf("container stats: %w", err))
			return
		}
		defer res.Body.Close()

		dec := json.NewDecoder(res.Body)
		for {
			var s mcont.StatsResponse
			if err := dec.Decode(&s); err != nil {
				if !errors.Is(err, io.EOF) {
					yield(container.Stats{}, fmt.Errorf("container stats: %w", err))
				}
				return
			}
			if !yield(stats(s), nil) {
				return
			}
		}
	}
}

// stats converts a sample of a Linux container. The daemon sends samples
// with only name and ID while the container isn't running.
func stats(in mcont.StatsResponse) container.Stats {
	if in.Read.IsZero() {
		return container.Stats{}
	}
	cpus := int(in.CPUStats.OnlineCPUs)
	if cpus == 0 {
		cpus = len(in.CPUStats.CPUUsage.PercpuUsage)
	}
	out := container.Stats{
		Read:       in.Read,
		CPUPercent: cpuPercent(in.CPUStats, in.PreCPUStats, cpus),
		OnlineCPUs: cpus,
		MemUsage:   memUsage(in.MemoryStats),
		MemLimit:   in.MemoryStats.Limit,
	}
	for _, n := range in.Networks {
		out.NetRx += n.RxBytes
		out.NetTx += n.TxBytes
	}
	return out
}

// cpuPercent is the container's share of the host's CPU time between the two
// samples, scaled to the number of CPUs like docker stats does. It is 0 for
// the first sample of a stream, which has no previous one.
func cpuPercent(cur, pre mcont.CPUStats, cpus int) float64 {
	if pre.SystemUsage == 0 || cur.CPUUsage.TotalUsage <= pre.CPUUsage.TotalUsage || cur.SystemUsage <= pre.SystemUsage {
		return 0
	}
	cpuDelta := float64(cur.CPUUsage.TotalUsage - pre.CPUUsage.TotalUsage)
	sysDelta := float64(cur.SystemUsage - pre.SystemUsage)
	return cpuDelta / sysDelta * float64(cpus) * 100
}

// memUsage leaves out the inactive page cache, which the kernel reclaims
// under pressure. cgroup v1 reports it as total_inactive_file, v2 as
// inactive_file.
func memUsage(m mcont.MemoryStats) uint64 {
	for _, k := range []string{"total_inactive_file", "inactive_file"} {
		if v, ok := m.Stats[k]; ok && v < m.Usage {
			return m.Usage - v
		}
	}
	return m.Usage
}
