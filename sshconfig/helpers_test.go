package sshconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// home is the fake home directory the test configs live in, under testdata/home/.ssh.
const home = "testdata/home"

func decodeTestdata(t *testing.T, name string) (*Config, error) {
	t.Helper()
	f, err := os.Open(filepath.Join(home, ".ssh", name)) //nolint:gosec // G304: test fixture
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	return Decode(f, WithHome(home))
}

// lookup decodes the testdata config name and looks up alias in it.
func lookup(t *testing.T, name, alias string) Values {
	t.Helper()
	cfg, err := decodeTestdata(t, name)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Lookup(alias)
}

// wantParseError checks that err is a *ParseError for file and line whose message contains
// want.
func wantParseError(t *testing.T, err error, file string, line int, want string) {
	t.Helper()
	perr, ok := errors.AsType[*ParseError](err)
	if !ok {
		t.Fatalf("error = %v, want a *ParseError", err)
	}
	if perr.File != file || perr.Line != line || !strings.Contains(perr.Err.Error(), want) {
		t.Errorf("error = %q, want file %q, line %d, containing %q", perr, file, line, want)
	}
}

// local is the local side used for token expansion in tests.
var local = Local{User: "max", Home: "/home/max", UID: "1000", Hostname: "laptop.example.org"}
