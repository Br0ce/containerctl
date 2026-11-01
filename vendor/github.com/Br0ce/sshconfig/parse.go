package sshconfig

import (
	"errors"
	"fmt"
	"strings"
)

// splitLine splits a config line into its lowercased keyword and arguments. The keyword
// may be separated from the arguments by whitespace or a single '='. It returns an empty
// keyword for blank lines and comments.
func splitLine(s string) (string, []string, error) {
	s = strings.TrimLeft(s, " \t")
	if s == "" || s[0] == '#' {
		return "", nil, nil
	}
	i := strings.IndexAny(s, " \t=")
	if i < 0 {
		return strings.ToLower(s), nil, nil
	}
	key := strings.ToLower(s[:i])
	rest := strings.TrimLeft(s[i:], " \t")
	rest = strings.TrimPrefix(rest, "=")
	args, err := splitArgs(rest)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", key, err)
	}
	return key, args, nil
}

// splitArgs splits s into whitespace separated arguments. Single or double quotes group an
// argument containing whitespace, a backslash escapes a quote, backslash or space, and an
// argument starting with '#' starts a comment.
func splitArgs(s string) ([]string, error) {
	var (
		args   []string
		cur    strings.Builder
		inWord bool
		quote  byte
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && strings.IndexByte(`\"' `, s[i+1]) >= 0:
			i++
			cur.WriteByte(s[i])
			inWord = true
		case quote != 0:
			if c == quote {
				quote = 0
				continue
			}
			cur.WriteByte(c)
		case c == '"' || c == '\'':
			quote = c
			inWord = true
		case c == ' ' || c == '\t':
			if inWord {
				args = append(args, cur.String())
				cur.Reset()
				inWord = false
			}
		case c == '#' && !inWord:
			return args, nil
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if inWord {
		args = append(args, cur.String())
	}
	return args, nil
}
