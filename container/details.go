package container

import "time"

// Details is the inspect information of a single container. Zero values mean
// the daemon didn't report the field or it isn't set.
type Details struct {
	ID      string
	Name    string
	Image   string
	ImageID string
	Created time.Time

	State      string
	Pid        int
	ExitCode   int
	OOMKilled  bool
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time

	// RestartPolicy is e.g. "always" or "on-failure:5".
	RestartPolicy string
	RestartCount  int

	Entrypoint []string
	Cmd        []string
	WorkingDir string
	User       string

	// Health is nil if the container has no healthcheck.
	Health *Health

	Networks []Network
	// Ports holds the exposed ports; Public is 0 for those not published on
	// the host.
	Ports  []Port
	Mounts []Mount

	Resources Resources

	// Env holds the environment as KEY=value pairs.
	Env    []string
	Labels map[string]string
}

// Health is the state of a container's healthcheck.
type Health struct {
	Status        string
	FailingStreak int
	// Checks holds the most recent checks, oldest first.
	Checks []HealthCheck
}

// HealthCheck is the result of a single healthcheck run.
type HealthCheck struct {
	Start    time.Time
	ExitCode int
	Output   string
}

// Network is a container's endpoint in a Docker network.
type Network struct {
	Name    string
	IP      string
	Gateway string
	MAC     string
	Aliases []string
}

// Mount is a volume, bind or tmpfs mounted into the container.
type Mount struct {
	Type        string
	Source      string
	Destination string
	RW          bool
}

// Resources are the limits set on a container; 0 means unlimited.
type Resources struct {
	Memory    int64 // bytes
	NanoCPUs  int64
	CPUShares int64
	PidsLimit int64
	Ulimits   []Ulimit
}

// Ulimit is a resource limit of the container's processes.
type Ulimit struct {
	Name       string
	Soft, Hard int64
}
