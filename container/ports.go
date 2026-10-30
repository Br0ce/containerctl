package container

import (
	"strconv"
	"strings"
)

// FormatPorts renders ports as host→container, e.g. "8888→8080, 53/udp". The
// container port is left out if it equals the host port, the protocol unless
// it is udp or sctp.
func FormatPorts(ports []Port) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		part := strconv.Itoa(int(p.Public))
		if p.Private != p.Public {
			part += "→" + strconv.Itoa(int(p.Private))
		}
		if p.Proto != "tcp" {
			part += "/" + p.Proto
		}
		parts[i] = part
	}
	return strings.Join(parts, ", ")
}
