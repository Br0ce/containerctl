package container

import "testing"

func TestFormatPorts(t *testing.T) {
	ports := []Port{
		{Public: 53, Private: 53, Proto: "udp"},
		{Public: 5432, Private: 5432, Proto: "tcp"},
		{Public: 8888, Private: 8080, Proto: "tcp"},
	}
	if got, want := FormatPorts(ports), "53/udp, 5432, 8888→8080"; got != want {
		t.Errorf("FormatPorts = %q, want %q", got, want)
	}
	if got := FormatPorts(nil); got != "" {
		t.Errorf("FormatPorts(nil) = %q, want empty", got)
	}
}
