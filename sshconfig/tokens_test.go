package sshconfig

import (
	"strings"
	"testing"
)

func TestLocalEnv(t *testing.T) {
	l, err := LocalEnv()
	if err != nil {
		t.Fatal(err)
	}
	if l.User == "" || l.Home == "" || l.UID == "" || l.Hostname == "" {
		t.Errorf("LocalEnv() = %+v, want all fields set", l)
	}
}

func TestExpandTokens(t *testing.T) {
	tokens := map[byte]string{'h': "web", 'p': "22"}
	tests := []struct {
		s, want, wantErr string
	}{
		{"example.com", "example.com", ""},
		{"%h.example.com", "web.example.com", ""},
		{"%h:%p", "web:22", ""},
		{"100%%", "100%", ""},
		{"%x", "", "unknown token %x"},
		{"web%", "", "trailing %"},
	}
	for _, tt := range tests {
		got, err := ExpandTokens(tt.s, tokens)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExpandTokens(%q) error = %v, want %q", tt.s, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ExpandTokens(%q) = %q, %v, want %q", tt.s, got, err, tt.want)
		}
	}
}
