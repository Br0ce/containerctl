package sshconfig

import "testing"

func TestMatchPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		want     bool
	}{
		{"web", []string{"web"}, true},
		{"web", []string{"WEB"}, true},
		{"web", []string{"db"}, false},
		{"web", []string{"*"}, true},
		{"web01", []string{"web*"}, true},
		{"web01", []string{"*01"}, true},
		{"web01", []string{"w*b*1"}, true},
		{"web1", []string{"web?"}, true},
		{"web10", []string{"web?"}, false},
		{"web", []string{"db", "web"}, true},
		{"web", []string{"*", "!web"}, false},
		{"web", []string{"!web", "*"}, false},
		{"db", []string{"*", "!web"}, true},
		{"db", []string{"!web"}, false},
	}
	for _, tt := range tests {
		if got := matchPatterns(tt.name, tt.patterns); got != tt.want {
			t.Errorf("matchPatterns(%q, %q) = %v, want %v", tt.name, tt.patterns, got, tt.want)
		}
	}
}
