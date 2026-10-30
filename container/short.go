package container

type Short struct {
	ID string
	// Project is the compose project, empty if the container wasn't started
	// by compose.
	Project string
	Name    string
	Image   string
	Ports   []Port
	Status  string
	Age     string
	State   string
	// Health is "healthy", "unhealthy", "starting" or "none" if the container
	// has no healthcheck. It is empty if the daemon doesn't report health.
	Health string
}

// Port is a container port published on the host.
type Port struct {
	Public  uint16
	Private uint16
	Proto   string
}
