package sshconfig

import (
	"maps"
	"strings"
)

// Values are the values that apply to a host alias, keyed by lowercased keyword. Get and
// GetAll return them expanded, GetRaw and GetAllRaw as written in the config.
type Values struct {
	alias  string
	values map[string][]string
	set    map[string]bool // keys given by Set, which are not expanded
	local  *localSide
}

// LookupOption configures Lookup.
type LookupOption func(*lookupConfig)

type lookupConfig struct {
	user, system *Config
	local        *Local
}

// WithUserConfig sets the user's config, e.g. from UserConfig, or a config read with
// DecodeFile to resolve like ssh -F, see DecodeFile for relative Include paths.
func WithUserConfig(c *Config) LookupOption {
	return func(lc *lookupConfig) {
		lc.user = c
	}
}

// WithSystemConfig sets the system-wide config, e.g. from SystemConfig.
func WithSystemConfig(c *Config) LookupOption {
	return func(lc *lookupConfig) {
		lc.system = c
	}
}

// WithLocalAs sets the local side of the connection, used to expand ~ and tokens such as
// %u, to l instead of the current user and host, e.g. for tests or to resolve another user's
// config. Values that need a field left empty in l make Get fail, so with Local{} nothing is
// taken from the local side.
func WithLocalAs(l Local) LookupOption {
	return func(lc *lookupConfig) {
		lc.local = &l
	}
}

// Lookup returns the values that apply when connecting to alias. As in OpenSSH, the user
// config is read before the system-wide one, the first value obtained in either wins, and a
// HostName from the user config applies to Match host in the system-wide one.
//
// The local side of the connection is the current user and host, as for ssh, unless
// WithLocalAs is given. It is only determined once a value needs it, and if that fails, Get
// returns the error for such values.
func Lookup(alias string, opts ...LookupOption) Values {
	var lc lookupConfig
	for _, opt := range opts {
		opt(&lc)
	}
	r := resolver{alias: alias, values: make(map[string][]string)}
	for _, c := range []*Config{lc.user, lc.system} {
		if c != nil {
			r.walk(c)
		}
	}
	local := &localSide{read: localEnv}
	if lc.local != nil {
		l := *lc.local
		local.read = func() (Local, error) { return l, nil }
	}
	return Values{alias: alias, values: r.values, local: local}
}

// Set returns a copy of vs with key set to v, replacing the values from the config, like an
// option given on the ssh command line. Get returns v as is, and tokens derived from key,
// e.g. %p for Port or %r for User, use v. Keys are case-insensitive.
func (vs Values) Set(key, v string) Values {
	key = strings.ToLower(key)
	values := make(map[string][]string, len(vs.values)+1)
	maps.Copy(values, vs.values)
	values[key] = []string{v}
	set := make(map[string]bool, len(vs.set)+1)
	maps.Copy(set, vs.set)
	set[key] = true
	vs.values, vs.set = values, set
	return vs
}

// Get returns the value in effect for key, i.e. the first one obtained, with ~ and the
// tokens key accepts expanded as OpenSSH does, see TOKENS in ssh_config(5). Tokens derived
// from other values use their expanded values: %h is the HostName or the alias, %p the Port
// or 22, %r the User or the local user. It returns "" if key is not set, and an error for a
// token key does not accept or a local value that is not known. Keys are case-insensitive.
func (vs Values) Get(key string) (string, error) {
	v := vs.GetRaw(key)
	if v == "" {
		return "", nil
	}
	return vs.expand(strings.ToLower(key), v)
}

// GetAll returns all values obtained for key in config order, e.g. for IdentityFile which
// may be given several times, expanded like Get.
func (vs Values) GetAll(key string) ([]string, error) {
	values := vs.GetAllRaw(key)
	if values == nil {
		return nil, nil
	}
	expanded := make([]string, len(values))
	for i, v := range values {
		var err error
		if expanded[i], err = vs.expand(strings.ToLower(key), v); err != nil {
			return nil, err
		}
	}
	return expanded, nil
}

// GetRaw returns the value in effect for key as written in the config, or "" if key is not
// set. Keys are case-insensitive.
func (vs Values) GetRaw(key string) string {
	if v := vs.values[strings.ToLower(key)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// GetAllRaw returns all values obtained for key in config order as written in the config.
// Keys are case-insensitive.
func (vs Values) GetAllRaw(key string) []string {
	return vs.values[strings.ToLower(key)]
}
