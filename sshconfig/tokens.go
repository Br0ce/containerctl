package sshconfig

import (
	"crypto/sha1" //nolint:gosec // G505: %C is a SHA-1 hash in OpenSSH
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"strings"
)

// Local describes the local side of a connection for token expansion.
type Local struct {
	User     string // %u, and %r if no User is set
	Home     string // %d, and ~ in paths
	UID      string // %i
	Hostname string // %l, and up to the first '.' %L
}

// LocalEnv returns the current user and host.
func LocalEnv() (Local, error) {
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

// expand expands the value v of the lowercased keyword key.
func (vs Values) expand(key, v string, local Local) (string, error) {
	if tildeKeywords[key] {
		v = expandTilde(key, v, local.Home)
	}
	allowed, ok := keywordTokens[key]
	if !ok {
		return v, nil
	}
	v, err := expand(v, func(c byte) (string, error) {
		if strings.IndexByte(allowed, c) < 0 {
			return "", fmt.Errorf("token %%%c not supported", c)
		}
		if key == "hostname" {
			return vs.alias, nil // only %h
		}
		return vs.token(c, local)
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

// expandTilde replaces a leading ~ in v, or in each of its paths for UserKnownHostsFile,
// with home.
func expandTilde(key, v, home string) string {
	tilde := func(p string) string {
		if p == "~" || strings.HasPrefix(p, "~/") {
			return home + p[1:]
		}
		return p
	}
	if key != "userknownhostsfile" {
		return tilde(v)
	}
	paths := strings.Split(v, " ")
	for i, p := range paths {
		paths[i] = tilde(p)
	}
	return strings.Join(paths, " ")
}

// token returns the value of the token c.
func (vs Values) token(c byte, local Local) (string, error) {
	switch c {
	case 'C':
		return vs.connectionHash(local)
	case 'd':
		return local.Home, nil
	case 'h':
		v, err := vs.Expand("hostname", local)
		if err != nil || v != "" {
			return v, err
		}
		return vs.alias, nil
	case 'i':
		return local.UID, nil
	case 'j':
		return vs.proxyJump(), nil
	case 'k':
		if v := vs.Get("hostkeyalias"); v != "" {
			return v, nil
		}
		return vs.alias, nil
	case 'L':
		short, _, _ := strings.Cut(local.Hostname, ".")
		return short, nil
	case 'l':
		return local.Hostname, nil
	case 'n':
		return vs.alias, nil
	case 'p':
		if v := vs.Get("port"); v != "" {
			return v, nil
		}
		return "22", nil
	case 'r':
		v, err := vs.Expand("user", local)
		if err != nil || v != "" {
			return v, err
		}
		return local.User, nil
	case 'u':
		return local.User, nil
	}
	return "", fmt.Errorf("token %%%c not supported", c)
}

// proxyJump returns the ProxyJump in effect, or "" if it is unset or none.
func (vs Values) proxyJump() string {
	if v := vs.Get("proxyjump"); !strings.EqualFold(v, "none") {
		return v
	}
	return ""
}

// connectionHash returns %C, the SHA-1 hash of %l%h%p%r%j in hex.
func (vs Values) connectionHash(local Local) (string, error) {
	var b strings.Builder
	b.WriteString(local.Hostname)
	for _, c := range []byte("hpr") {
		v, err := vs.token(c, local)
		if err != nil {
			return "", err
		}
		b.WriteString(v)
	}
	b.WriteString(vs.proxyJump())
	sum := sha1.Sum([]byte(b.String())) //nolint:gosec // G401: %C is a SHA-1 hash in OpenSSH
	return hex.EncodeToString(sum[:]), nil
}

// ExpandTokens replaces the %-tokens in s, e.g. %h, with their values from tokens and %%
// with a single %. An unknown token or a trailing % is an error. Values.Expand knows which
// tokens each keyword accepts and derives their values.
func ExpandTokens(s string, tokens map[byte]string) (string, error) {
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
