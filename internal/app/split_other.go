//go:build !windows

package app

import (
	"errors"
	"strings"
)

// splitCommand splits an editor command as a POSIX shell would for words,
// single and double quotes and backslashes; no expansion (ui-design.md,
// $EDITOR, item 1).
func splitCommand(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	in, quote, esc := false, byte(0), false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case esc:
			cur.WriteByte(c)
			esc = false
		case quote == '\'':
			if c == '\'' {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
		case c == '\\' && quote != '\'':
			esc, in = true, true
		case quote == '"':
			if c == '"' {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
		case c == '\'' || c == '"':
			quote, in = c, true
		case c == ' ' || c == '\t' || c == '\n':
			if in {
				args, in = append(args, cur.String()), false
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
			in = true
		}
	}
	if quote != 0 || esc {
		return nil, errors.New("unterminated quote or escape")
	}
	if in {
		args = append(args, cur.String())
	}
	return args, nil
}
