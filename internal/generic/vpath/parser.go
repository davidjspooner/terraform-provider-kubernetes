package vpath

import (
	"strings"
	"text/scanner"
)

func MustCompile(s string) Path {
	p, err := Compile(s)
	if err != nil {
		panic(err)
	}
	return p
}

func Compile(text string) (Path, error) {
	var s scanner.Scanner

	if len(text) > 0 && (text[0] != '.') && (text[0] != '[') {
		text = "." + text
	}

	s.Init(strings.NewReader(text))
	s.Mode ^= scanner.SkipComments // don't skip comments
	s.Whitespace = 0
	var path Path
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case '.':
			if tok = s.Scan(); tok != scanner.Ident {
				if tok == scanner.EOF && len(path) == 0 {
					return path, nil
				}

				return nil, failedExpectation(text, "identifier", &s)
			}
			f := Field(s.TokenText())
			path = append(path, &f)
		case '[':
			tok = s.Scan()
			prefix := ""
			if tok == '-' {
				prefix = "-"
				tok = s.Scan()
			}
			if tok != scanner.Int {
				return nil, failedExpectation(text, "integer", &s)
			}
			firstIndex := prefix + s.TokenText()
			tok = s.Scan()
			if tok != ']' {
				return nil, failedExpectation(text, "]", &s)
			}
			f := Field(firstIndex)
			path = append(path, &f)
		default:
			return nil, failedExpectation(text, ". or [", &s)
		}
	}
	return path, nil
}

type ParseError struct {
	Path     string
	Expected string
	Actual   string
}

func (e *ParseError) Error() string {
	return "invalid path '" + e.Path + "': expected " + e.Expected + ", got " + e.Actual
}

func failedExpectation(text, expected string, s *scanner.Scanner) error {
	e := ParseError{
		Path:     text,
		Expected: expected,
		Actual:   s.TokenText(),
	}
	return &e
}
