package cmdexec

import (
	"fmt"
	"strings"
	"unicode"
)

type ParsedCommand struct {
	Program  string
	Args     []string
	Redirect *Redirect
}

type Redirect struct {
	Target string
	Append bool
}

// Parse tokenizes a command line without invoking a shell.
// Handles single/double quotes and > / >> redirection for echo.
func Parse(cmdLine string) (*ParsedCommand, error) {
	cmdLine = strings.TrimSpace(cmdLine)
	if cmdLine == "" {
		return nil, fmt.Errorf("empty command")
	}

	tokens, redirect, err := tokenize(cmdLine)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command after parsing")
	}

	return &ParsedCommand{
		Program:  tokens[0],
		Args:     tokens[1:],
		Redirect: redirect,
	}, nil
}

func tokenize(input string) ([]string, *Redirect, error) {
	var tokens []string
	var current strings.Builder
	var redirect *Redirect
	inSingle := false
	inDouble := false
	i := 0

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for i < len(input) {
		ch := rune(input[i])

		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				current.WriteByte(input[i])
			}
			i++
			continue
		}

		if inDouble {
			if ch == '"' {
				inDouble = false
			} else {
				current.WriteByte(input[i])
			}
			i++
			continue
		}

		switch {
		case ch == '\'':
			inSingle = true
			i++
		case ch == '"':
			inDouble = true
			i++
		case ch == '>' && redirect == nil:
			flush()
			append_ := false
			i++
			if i < len(input) && input[i] == '>' {
				append_ = true
				i++
			}
			// skip whitespace after >
			for i < len(input) && unicode.IsSpace(rune(input[i])) {
				i++
			}
			// parse redirect target
			target, err := parseRedirectTarget(input[i:])
			if err != nil {
				return nil, nil, err
			}
			redirect = &Redirect{Target: target, Append: append_}
			i += len(target)
			// skip trailing whitespace
			for i < len(input) && unicode.IsSpace(rune(input[i])) {
				i++
			}
		case unicode.IsSpace(ch):
			flush()
			i++
		default:
			current.WriteByte(input[i])
			i++
		}
	}

	if inSingle || inDouble {
		return nil, nil, fmt.Errorf("unclosed quote")
	}

	flush()
	return tokens, redirect, nil
}

func parseRedirectTarget(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("missing redirect target")
	}

	var target strings.Builder
	i := 0
	inSingle := false
	inDouble := false

	for i < len(s) {
		ch := rune(s[i])
		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				target.WriteByte(s[i])
			}
			i++
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			} else {
				target.WriteByte(s[i])
			}
			i++
			continue
		}
		if ch == '\'' {
			inSingle = true
			i++
			continue
		}
		if ch == '"' {
			inDouble = true
			i++
			continue
		}
		if unicode.IsSpace(ch) {
			break
		}
		target.WriteByte(s[i])
		i++
	}

	if target.Len() == 0 {
		return "", fmt.Errorf("missing redirect target")
	}
	return target.String(), nil
}
