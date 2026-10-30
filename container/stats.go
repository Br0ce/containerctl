package container

import "time"

// Stats is one sample of a container's resource usage. Read is zero if the
// container isn't running, in which case all other fields are zero too.
type Stats struct {
	Read time.Time

	// CPUPercent is the CPU usage since the previous sample, where 100 is
	// one full CPU; it goes up to 100 times OnlineCPUs.
	CPUPercent float64
	OnlineCPUs int

	// MemUsage excludes the page cache, like docker stats does.
	MemUsage uint64 // bytes
	MemLimit uint64 // bytes

	// NetRx and NetTx are the bytes received and sent since the container
	// started, summed over all interfaces.
	NetRx uint64
	NetTx uint64
}
