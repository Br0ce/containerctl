package sshconfig

import (
	"maps"
	"strings"
)

// Values are the values that apply to a host alias, keyed by lowercased keyword. Get and
// GetAll return them as written in the config, Expand and ExpandAll expand them.
type Values struct {
	alias  string
	values map[string][]string
	set    map[string]bool // keys given by Set, which are not expanded
}

// Lookup returns the values that apply when connecting to alias, reading configs in order
// like OpenSSH reads the user config before the system-wide one. The first value obtained in
// any of them wins, and a HostName from an earlier config applies to Match host in a later.
func Lookup(alias string, configs ...*Config) Values {
	r := resolver{alias: alias, values: make(map[string][]string)}
	for _, c := range configs {
		r.walk(c)
	}
	return Values{alias: alias, values: r.values}
}

// Set returns a copy of vs with key set to v, replacing the values from the config, like an
// option given on the ssh command line. Expand returns v as is, and tokens derived from key,
// e.g. %p for Port or %r for User, use v. Keys are case-insensitive.
func (vs Values) Set(key, v string) Values {
	key = strings.ToLower(key)
	values := make(map[string][]string, len(vs.values)+1)
	maps.Copy(values, vs.values)
	values[key] = []string{v}
	set := make(map[string]bool, len(vs.set)+1)
	maps.Copy(set, vs.set)
	set[key] = true
	return Values{alias: vs.alias, values: values, set: set}
}

// Get returns the value in effect for key, i.e. the first one obtained, or "" if key is
// not set. Keys are case-insensitive.
func (vs Values) Get(key string) string {
	if v := vs.values[strings.ToLower(key)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// GetAll returns all values obtained for key in config order, e.g. for IdentityFile which
// may be given several times. Keys are case-insensitive.
func (vs Values) GetAll(key string) []string {
	return vs.values[strings.ToLower(key)]
}

// Expand returns the value in effect for key, like Get, with ~ and the tokens key accepts
// expanded as OpenSSH does, see TOKENS in ssh_config(5). Tokens derived from other values
// use their expanded values: %h is the HostName or the alias, %p the Port or 22, %r the User
// or local.User. It returns "" if key is not set, and an error for a token key does not
// accept.
func (vs Values) Expand(key string, local Local) (string, error) {
	v := vs.Get(key)
	if v == "" {
		return "", nil
	}
	return vs.expand(strings.ToLower(key), v, local)
}

// ExpandAll returns all values obtained for key, like GetAll, expanded like Expand.
func (vs Values) ExpandAll(key string, local Local) ([]string, error) {
	values := vs.GetAll(key)
	if values == nil {
		return nil, nil
	}
	expanded := make([]string, len(values))
	for i, v := range values {
		var err error
		if expanded[i], err = vs.expand(strings.ToLower(key), v, local); err != nil {
			return nil, err
		}
	}
	return expanded, nil
}
