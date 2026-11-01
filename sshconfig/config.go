package sshconfig

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// defaultMaxIncludeDepth limits nested Include directives, like OpenSSH's
// READCONF_MAX_DEPTH.
const defaultMaxIncludeDepth = 16

// Config is a parsed ssh config file with its Include directives already read.
type Config struct {
	home            string
	includeDir      string
	maxIncludeDepth int
	lines           []line
}

type line struct {
	key      string // lowercased keyword
	args     []string
	includes []*Config // only for the include keyword
}

// systemConfigPath returns the system-wide config read by SystemConfig.
func systemConfigPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "ssh", "ssh_config")
	}
	return "/etc/ssh/ssh_config"
}

// Option configures how a config is decoded.
type Option func(*Config)

// WithHome sets the user's home directory: Include paths starting with ~/ are resolved
// against it. It defaults to os.UserHomeDir.
func WithHome(dir string) Option {
	return func(c *Config) {
		c.home = dir
	}
}

// WithIncludeDir sets the directory relative Include paths are resolved against, e.g.
// /etc/ssh for the system-wide config. It defaults to the directory of the file for
// DecodeFile, UserConfig and SystemConfig, and to ~/.ssh for Decode.
func WithIncludeDir(dir string) Option {
	return func(c *Config) {
		c.includeDir = dir
	}
}

// WithMaxIncludeDepth sets how many levels of nested Include directives are read before
// decoding fails, 16 by default as in OpenSSH. With 0, or a negative n, any Include is an
// error.
func WithMaxIncludeDepth(n int) Option {
	return func(c *Config) {
		c.maxIncludeDepth = max(n, 0)
	}
}

// newConfig returns an empty config with opts applied and the home directory set.
func newConfig(opts []Option) (*Config, error) {
	c := &Config{maxIncludeDepth: defaultMaxIncludeDepth}
	for _, opt := range opts {
		opt(c)
	}
	if c.home == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home directory: %w", err)
		}
		c.home = home
	}
	return c, nil
}

// Decode parses the ssh config read from r. Included files that do not exist or cannot be
// read are skipped, like OpenSSH does. An invalid line is reported as a *ParseError.
func Decode(r io.Reader, opts ...Option) (*Config, error) {
	c, err := newConfig(opts)
	if err != nil {
		return nil, err
	}
	if c.includeDir == "" {
		c.includeDir = filepath.Join(c.home, ".ssh")
	}
	return c.decode(r, "", 0)
}

// DecodeFile parses the ssh config file at path. If it does not exist, the error wraps
// fs.ErrNotExist.
func DecodeFile(path string, opts ...Option) (*Config, error) {
	c, err := newConfig(opts)
	if err != nil {
		return nil, err
	}
	return c.decodePath(path)
}

// UserConfig parses the user's config, ~/.ssh/config. A missing file yields an empty config.
func UserConfig(opts ...Option) (*Config, error) {
	c, err := newConfig(opts)
	if err != nil {
		return nil, err
	}
	return c.decodeOptional(filepath.Join(c.home, ".ssh", "config"))
}

// SystemConfig parses the system-wide config, /etc/ssh/ssh_config or
// %ProgramData%\ssh\ssh_config on Windows. A missing file yields an empty config.
func SystemConfig(opts ...Option) (*Config, error) {
	c, err := newConfig(opts)
	if err != nil {
		return nil, err
	}
	return c.decodeOptional(systemConfigPath())
}

// decodeOptional parses the file at path into c, leaving c empty if it does not exist.
func (c *Config) decodeOptional(path string) (*Config, error) {
	cfg, err := c.decodePath(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	return cfg, err
}

// Lookup returns the values that apply when connecting to alias.
func (c *Config) Lookup(alias string) Values {
	return Lookup(alias, c)
}

// decodePath parses the file at path into c. Relative includes resolve against the file's
// directory unless WithIncludeDir was given.
func (c *Config) decodePath(path string) (*Config, error) {
	f, err := os.Open(path) //nolint:gosec // G304: reading the named config is intended
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if c.includeDir == "" {
		c.includeDir = filepath.Dir(path)
	}
	return c.decode(f, path, 0)
}

// decode parses r, read from the file name, into c, reading Include directives at the given
// nesting depth.
func (c *Config) decode(r io.Reader, name string, depth int) (*Config, error) {
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		key, args, err := splitLine(sc.Text())
		if err != nil {
			return nil, &ParseError{File: name, Line: n, Err: err}
		}
		if key == "" {
			continue
		}
		if len(args) == 0 {
			return nil, &ParseError{File: name, Line: n, Err: fmt.Errorf("%s: missing argument", key)}
		}
		l := line{key: key, args: args}
		if key == "include" {
			if l.includes, err = c.readIncludes(args, depth+1); err != nil {
				// Report the innermost line, i.e. the one in the included file.
				if _, ok := errors.AsType[*ParseError](err); ok {
					return nil, err
				}
				return nil, &ParseError{File: name, Line: n, Err: err}
			}
		}
		c.lines = append(c.lines, l)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read ssh config: %w", err)
	}
	return c, nil
}

// readIncludes reads all files matching the glob patterns of an Include directive.
func (c *Config) readIncludes(patterns []string, depth int) ([]*Config, error) {
	if depth > c.maxIncludeDepth {
		return nil, errors.New("include: too many nested includes")
	}
	var includes []*Config
	for _, pattern := range patterns {
		switch {
		case strings.HasPrefix(pattern, "~/"):
			pattern = filepath.Join(c.home, pattern[2:])
		case !filepath.IsAbs(pattern):
			pattern = filepath.Join(c.includeDir, pattern)
		}
		// Glob returns the matches sorted, the same order OpenSSH reads them in.
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("include %s: %w", pattern, err)
		}
		for _, name := range matches {
			inc, err := c.readInclude(name, depth)
			if err != nil {
				return nil, err
			}
			if inc != nil {
				includes = append(includes, inc)
			}
		}
	}
	return includes, nil
}

// readInclude parses an included file. It returns nil if the file does not exist, cannot be
// read or is a directory.
func (c *Config) readInclude(name string, depth int) (*Config, error) {
	f, err := os.Open(name) //nolint:gosec // G304: reading the files named by Include is intended, like OpenSSH
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("include: %w", err)
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.IsDir() {
		return nil, nil
	}

	inc := &Config{home: c.home, includeDir: c.includeDir, maxIncludeDepth: c.maxIncludeDepth}
	return inc.decode(f, name, depth)
}
