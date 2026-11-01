package sshconfig

import "strings"

type resolver struct {
	alias  string
	values map[string][]string
}

// walk collects the values of all active lines of c. Each file starts active; a Host or
// Match line decides whether the following lines apply up to the next one in the same
// file. An Include is only read if the line it is on is active.
func (r *resolver) walk(c *Config) {
	active := true
	for _, l := range c.lines {
		switch l.key {
		case "host":
			active = matchPatterns(r.alias, l.args)
		case "match":
			active = r.match(l.args)
		case "include":
			if active {
				for _, inc := range l.includes {
					r.walk(inc)
				}
			}
		default:
			if active {
				r.values[l.key] = append(r.values[l.key], strings.Join(l.args, " "))
			}
		}
	}
}

// match evaluates the criteria of a Match line, all of which must hold. A criterion can be
// negated with '!'. Unsupported criteria make the line never match.
func (r *resolver) match(args []string) bool {
	for i := 0; i < len(args); i++ {
		criterion, negated := strings.CutPrefix(strings.ToLower(args[i]), "!")
		var ok bool
		switch criterion {
		case "all":
			ok = true
		case "host", "originalhost":
			if i+1 >= len(args) {
				return false
			}
			i++
			name := r.alias
			if criterion == "host" {
				name = r.hostname()
			}
			ok = matchPatterns(name, strings.Split(args[i], ","))
		default:
			return false
		}
		if ok == negated {
			return false
		}
	}
	return true
}

// hostname returns the target host as it stands so far: the first HostName with %h
// expanded, or the alias if none is set yet.
func (r *resolver) hostname() string {
	if v := r.values["hostname"]; len(v) > 0 {
		if host, err := expandTokens(v[0], map[byte]string{'h': r.alias}); err == nil {
			return host
		}
	}
	return r.alias
}

// matchPatterns reports whether name matches the pattern list: at least one pattern must
// match and no negated pattern (prefixed with '!') may match. Matching is case-insensitive.
func matchPatterns(name string, patterns []string) bool {
	name = strings.ToLower(name)
	matched := false
	for _, p := range patterns {
		p = strings.ToLower(p)
		if neg, ok := strings.CutPrefix(p, "!"); ok {
			if wildcard(neg, name) {
				return false
			}
			continue
		}
		if wildcard(p, name) {
			matched = true
		}
	}
	return matched
}

// wildcard reports whether s matches pattern, where '*' matches any sequence of characters
// and '?' exactly one.
func wildcard(pattern, s string) bool {
	for pattern != "" {
		switch pattern[0] {
		case '*':
			pattern = strings.TrimLeft(pattern, "*")
			if pattern == "" {
				return true
			}
			for i := range len(s) + 1 {
				if wildcard(pattern, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if s == "" {
				return false
			}
		default:
			if s == "" || s[0] != pattern[0] {
				return false
			}
		}
		pattern, s = pattern[1:], s[1:]
	}
	return s == ""
}
