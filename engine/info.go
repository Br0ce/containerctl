package engine

// Info describes the Docker host the client is connected to.
type Info struct {
	Hostname   string
	Version    string
	APIVersion string
	OS         string
	Arch       string
	CPUs       int
	Memory     int64 // bytes
}
