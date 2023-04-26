package main

import (
	"io"
	"regexp"
	"strings"
)

var (
	bulletRegexp   = regexp.MustCompile(`^\s*([-*]|\d+\.) `)
	quoteRegexp    = regexp.MustCompile(`^> `)
	wordlikeRegexp = regexp.MustCompile(`^\s*\S*`)
)

// TODO: Use rune length rather than byte length
type MarkdownWriter struct {
	maxLen    int
	prefix    string
	lineLen   int
	output    io.StringWriter
	wrap      bool
	inputLine strings.Builder
}

func NewMarkdownWriter(output io.StringWriter, maxLineLen int) *MarkdownWriter {
	return &MarkdownWriter{
		maxLen: maxLineLen,
		output: output,
		wrap:   true,
	}
}

func (mdw *MarkdownWriter) MaxLen() int {
	return mdw.maxLen
}

func (mdw *MarkdownWriter) WriteString(str string) (int, error) {
	var line string
	var rest string = str

	for len(rest) > 0 {
		var found bool
		line, rest, found = strings.Cut(rest, "\n")
		mdw.inputLine.WriteString(line)

		if mdw.wrap {
			mdw.wrapLine(line)
		} else {
			mdw.output.WriteString(line)
		}

		if found {
			// TODO: Improve this (it currently doesn't allow long strings after
			// '```', but maybe that's OK?)
			//
			// A very crude implementation to avoid clobbering code
			if mdw.inputLine.Len() == 0 && strings.HasPrefix(mdw.inputLine.String(), "```") {
				mdw.wrap = !mdw.wrap
			}

			mdw.output.WriteString("\n")
			mdw.lineLen = 0
			mdw.inputLine.Reset()
			mdw.prefix = ""
		}
	}

	// TODO: Return errors
	return len(str), nil
}

func (mdw *MarkdownWriter) wrapLine(line string) {
	if str := bulletRegexp.FindString(mdw.inputLine.String()); str != "" {
		mdw.prefix = strings.Repeat(" ", len(str))
	} else if quoteRegexp.MatchString(mdw.inputLine.String()) {
		mdw.prefix = "> "
	}

	for len(line) > 0 {
		next := wordlikeRegexp.FindString(line)
		line = line[len(next):]

		if len(next)+mdw.lineLen > mdw.maxLen {
			next = strings.TrimSpace(next)

			if next != "" {
				mdw.output.WriteString("\n")
				mdw.output.WriteString(mdw.prefix)
				mdw.lineLen = len(mdw.prefix)
			}
		}

		mdw.output.WriteString(next)
		mdw.lineLen += len(next)
	}
}
