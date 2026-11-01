package sshconfig

import (
	"crypto/sha1" //nolint:gosec // G505: %C is a SHA-1 hash in OpenSSH
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"strings"
	"sync"
)

// Local describes the local side of a connection for token expansion, see WithLocalAs.
type Local struct {
	User     string // %u, and %r if no User is set
	Home     string // %d, and ~ in paths
	UID      string // %i
	Hostname string // %l, and up to the first '.' %L
}

// localEnv returns the current user and host. It is a variable so tests can count the
// calls.
var localEnv = func() (Local, error) {
	u, err := user.Current()
	if err != nil {
		return Local{}, fmt.Errorf("current user: %w", err)
	}
	host, err := os.Hostname()
	if err != nil {
		return Local{}, fmt.Errorf("hostname: %w", err)
	}
	return Local{User: u.Username, Home: u.HomeDir, UID: u.Uid, Hostname: host}, nil
}

// commonTokens are the tokens most keywords accept, see TOKENS in ssh_config(5).
const commonTokens = "CdhijkLlnpru" //nolint:gosec // G101: token letters, not a credential

// keywordTokens lists the tokens each keyword accepts besides %%. Keywords not listed take
// no tokens. Tokens only known while connecting, e.g. %f or %T, are not supported.
var keywordTokens = map[string]string{
	"certificatefile":    commonTokens,
	"controlpath":        commonTokens,
	"identityagent":      commonTokens,
	"identityfile":       commonTokens,
	"knownhostscommand":  commonTokens,
	"localcommand":       commonTokens,
	"localforward":       commonTokens,
	"remotecommand":      commonTokens,
	"remoteforward":      commonTokens,
	"revokedhostkeys":    commonTokens,
	"setenv":             commonTokens,
	"userknownhostsfile": commonTokens,
	"versionaddendum":    commonTokens,
	"hostname":           "h",
	"proxycommand":       "hnpr",
	"proxyjump":          "hnpr",
	"user":               "dhijkLlnpu",
}

// tildeKeywords are the keywords whose paths may start with ~ for the user's home directory.
var tildeKeywords = map[string]bool{
	"certificatefile":    true,
	"controlpath":        true,
	"identityagent":      true,
	"identityfile":       true,
	"revokedhostkeys":    true,
	"userknownhostsfile": true,
}

// localSide is the local side of a connection, read on first use and shared by the copies
// of a Values.
type localSide struct {
	read  func() (Local, error)
	once  sync.Once
	local Local
	err   error
}

func (ls *localSide) get() (Local, error) {
	if ls == nil {
		return Local{}, nil
	}
	ls.once.Do(func() {
		ls.local, ls.err = ls.read()
	})
	return ls.local, ls.err
}

// expand expands the value v of the lowercased keyword key.
func (vs Values) expand(key, v string) (string, error) {
	if vs.set[key] {
		return v, nil
	}
	var err error
	if tildeKeywords[key] {
		if v, err = vs.expandTilde(key, v); err != nil {
			return "", fmt.Errorf("%s: %w", key, err)
		}
	}
	allowed, ok := keywordTokens[key]
	if !ok {
		return v, nil
	}
	v, err = expand(v, func(c byte) (string, error) {
		if strings.IndexByte(allowed, c) < 0 {
			return "", fmt.Errorf("token %%%c not supported", c)
		}
		if key == "hostname" {
			return vs.alias, nil // only %h
		}
		return vs.token(c)
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

// expandTilde replaces a leading ~ in v, or in each of its paths for UserKnownHostsFile,
// with the local home directory.
func (vs Values) expandTilde(key, v string) (string, error) {
	paths := []string{v}
	if key == "userknownhostsfile" {
		paths = strings.Split(v, " ")
	}
	for i, p := range paths {
		if p != "~" && !strings.HasPrefix(p, "~/") {
			continue
		}
		home, err := vs.localValue("home directory", func(l Local) string { return l.Home })
		if err != nil {
			return "", fmt.Errorf("%q: ~: %w", v, err)
		}
		paths[i] = home + p[1:]
	}
	return strings.Join(paths, " "), nil
}

// localValue returns the field of the local side selected by field, described by name, or an
// error if it is not known.
func (vs Values) localValue(name string, field func(Local) string) (string, error) {
	l, err := vs.local.get()
	if err != nil {
		return "", err
	}
	if v := field(l); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("local %s not known, see WithLocalAs", name)
}

// token returns the value of the token c.
func (vs Values) token(c byte) (string, error) {
	switch c {
	case 'C':
		return vs.connectionHash()
	case 'd':
		return vs.localValue("home directory", func(l Local) string { return l.Home })
	case 'h':
		v, err := vs.Get("hostname")
		if err != nil || v != "" {
			return v, err
		}
		return vs.alias, nil
	case 'i':
		return vs.localValue("user ID", func(l Local) string { return l.UID })
	case 'j':
		return vs.proxyJump(), nil
	case 'k':
		if v := vs.GetRaw("hostkeyalias"); v != "" {
			return v, nil
		}
		return vs.alias, nil
	case 'L':
		host, err := vs.localValue("host name", func(l Local) string { return l.Hostname })
		short, _, _ := strings.Cut(host, ".")
		return short, err
	case 'l':
		return vs.localValue("host name", func(l Local) string { return l.Hostname })
	case 'n':
		return vs.alias, nil
	case 'p':
		if v := vs.GetRaw("port"); v != "" {
			return v, nil
		}
		return "22", nil
	case 'r':
		v, err := vs.Get("user")
		if err != nil || v != "" {
			return v, err
		}
		return vs.localValue("user", func(l Local) string { return l.User })
	case 'u':
		return vs.localValue("user", func(l Local) string { return l.User })
	}
	return "", fmt.Errorf("token %%%c not supported", c)
}

// proxyJump returns the ProxyJump in effect, or "" if it is unset or none.
func (vs Values) proxyJump() string {
	if v := vs.GetRaw("proxyjump"); !strings.EqualFold(v, "none") {
		return v
	}
	return ""
}

// connectionHash returns %C, the SHA-1 hash of %l%h%p%r%j in hex.
func (vs Values) connectionHash() (string, error) {
	var b strings.Builder
	for _, c := range []byte("lhpr") {
		v, err := vs.token(c)
		if err != nil {
			return "", err
		}
		b.WriteString(v)
	}
	b.WriteString(vs.proxyJump())
	sum := sha1.Sum([]byte(b.String())) //nolint:gosec // G401: %C is a SHA-1 hash in OpenSSH
	return hex.EncodeToString(sum[:]), nil
}

// expandTokens replaces the %-tokens in s, e.g. %h, with their values from tokens and %%
// with a single %. An unknown token or a trailing % is an error.
func expandTokens(s string, tokens map[byte]string) (string, error) {
	return expand(s, func(c byte) (string, error) {
		v, ok := tokens[c]
		if !ok {
			return "", fmt.Errorf("unknown token %%%c", c)
		}
		return v, nil
	})
}

// expand replaces the %-tokens in s with the values returned by token and %% with a single %.
func expand(s string, token func(c byte) (string, error)) (string, error) {
	if !strings.Contains(s, "%") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i == len(s) {
			return "", fmt.Errorf("%q: trailing %%", s)
		}
		if s[i] == '%' {
			b.WriteByte('%')
			continue
		}
		v, err := token(s[i])
		if err != nil {
			return "", fmt.Errorf("%q: %w", s, err)
		}
		b.WriteString(v)
	}
	return b.String(), nil
}
