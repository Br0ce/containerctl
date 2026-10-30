package client

import (
	"testing"
	"time"

	mcont "github.com/moby/moby/api/types/container"

	"github.com/Br0ce/containerctl/container"
)

func TestStats(t *testing.T) {
	read := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	in := mcont.StatsResponse{
		Read: read,
		CPUStats: mcont.CPUStats{
			CPUUsage:    mcont.CPUUsage{TotalUsage: 3_000},
			SystemUsage: 20_000,
			OnlineCPUs:  4,
		},
		PreCPUStats: mcont.CPUStats{
			CPUUsage:    mcont.CPUUsage{TotalUsage: 1_000},
			SystemUsage: 10_000,
		},
		MemoryStats: mcont.MemoryStats{
			Usage: 300 << 20,
			Limit: 1 << 30,
			Stats: map[string]uint64{"inactive_file": 100 << 20},
		},
		Networks: map[string]mcont.NetworkStats{
			"eth0": {RxBytes: 100, TxBytes: 10},
			"eth1": {RxBytes: 200, TxBytes: 20},
		},
	}
	want := container.Stats{
		Read:       read,
		CPUPercent: 80, // 2000 of 10000 ns on 4 CPUs
		OnlineCPUs: 4,
		MemUsage:   200 << 20,
		MemLimit:   1 << 30,
		NetRx:      300,
		NetTx:      30,
	}
	if got := stats(in); got != want {
		t.Errorf("stats() = %+v, want %+v", got, want)
	}
}

func TestStatsNotRunning(t *testing.T) {
	in := mcont.StatsResponse{ID: "aaaaaaaaaaaaaaaa", Name: "/web"}
	if got := stats(in); got != (container.Stats{}) {
		t.Errorf("stats() = %+v, want empty", got)
	}
}

func TestCPUPercent(t *testing.T) {
	cpu := func(total, system uint64) mcont.CPUStats {
		return mcont.CPUStats{CPUUsage: mcont.CPUUsage{TotalUsage: total}, SystemUsage: system}
	}
	tests := []struct {
		name     string
		cur, pre mcont.CPUStats
		want     float64
	}{
		{"busy", cpu(2_000, 4_000), cpu(1_000, 2_000), 100},
		{"first sample", cpu(2_000, 4_000), mcont.CPUStats{}, 0},
		{"idle", cpu(1_000, 4_000), cpu(1_000, 2_000), 0},
		{"no system delta", cpu(2_000, 2_000), cpu(1_000, 2_000), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cpuPercent(tt.cur, tt.pre, 2); got != tt.want {
				t.Errorf("cpuPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemUsage(t *testing.T) {
	tests := []struct {
		name  string
		stats map[string]uint64
		want  uint64
	}{
		{"cgroup v1", map[string]uint64{"total_inactive_file": 30}, 70},
		{"cgroup v2", map[string]uint64{"inactive_file": 40}, 60},
		{"no cache", nil, 100},
		{"cache above usage", map[string]uint64{"inactive_file": 200}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memUsage(mcont.MemoryStats{Usage: 100, Stats: tt.stats}); got != tt.want {
				t.Errorf("memUsage() = %d, want %d", got, tt.want)
			}
		})
	}
}
