// Package sshconfig reads an OpenSSH client configuration (see ssh_config(5)) and resolves
// the values that apply to a host alias.
//
// It supports Host blocks with wildcards and negation, Match blocks with the all, host and
// originalhost criteria, and Include directives. As in OpenSSH, the first value obtained for
// a keyword wins, also across several files, e.g. the user config followed by the
// system-wide one. Other Match criteria (e.g. exec, user, canonical) are not supported: a
// Match block using them never matches.
//
// UserConfig and SystemConfig read the standard files, and Lookup resolves an alias against
// them. Get returns a value with the token and tilde expansion OpenSSH does for its keyword,
// GetRaw as written in the config:
//
//	user, err := sshconfig.UserConfig()
//	...
//	system, err := sshconfig.SystemConfig()
//	...
//	vals := sshconfig.Lookup("web",
//		sshconfig.WithUserConfig(user),
//		sshconfig.WithSystemConfig(system),
//	)
//	identityFiles, err := vals.GetAll("IdentityFile")
package sshconfig
