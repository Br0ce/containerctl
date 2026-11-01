package sshconfig

import (
	"slices"
	"testing"
)

func TestSplitLine(t *testing.T) {
	tests := []struct {
		line     string
		wantKey  string
		wantArgs []string
	}{
		{"", "", nil},
		{"   # comment", "", nil},
		{"HostName example.com", "hostname", []string{"example.com"}},
		{"\tPort=22", "port", []string{"22"}},
		{"Port = 22", "port", []string{"22"}},
		{"Host a b  c", "host", []string{"a", "b", "c"}},
		{`IdentityFile "~/.ssh/my key"`, "identityfile", []string{"~/.ssh/my key"}},
		{`IdentityFile '~/.ssh/my key'`, "identityfile", []string{"~/.ssh/my key"}},
		{`IdentityFile ~/.ssh/my\ key`, "identityfile", []string{"~/.ssh/my key"}},
		{`Match exec "test -n \"$CI\"" host x`, "match", []string{"exec", `test -n "$CI"`, "host", "x"}},
		{"User bob # comment", "user", []string{"bob"}},
		{"User b#ob", "user", []string{"b#ob"}},
	}
	for _, tt := range tests {
		key, args, err := splitLine(tt.line)
		if err != nil {
			t.Errorf("splitLine(%q) error = %v", tt.line, err)
			continue
		}
		if key != tt.wantKey || !slices.Equal(args, tt.wantArgs) {
			t.Errorf("splitLine(%q) = %q, %q, want %q, %q", tt.line, key, args, tt.wantKey, tt.wantArgs)
		}
	}
}
