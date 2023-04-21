package main

import (
	"io"
	"regexp"
	"strings"
)

var bulletRegexp = regexp.MustCompile(`^\s*[-*] `)
var wordlikeRegexp = regexp.MustCompile(`^\s*\S*`)

type MarkdownWriter struct {
	maxLen  int
	prefix  string
	lineLen int
	output  io.StringWriter
	wrap    bool
}

func NewMarkdownWriter(output io.StringWriter, maxLineLen int) *MarkdownWriter {
	return &MarkdownWriter{
		maxLen: maxLineLen,
		output: output,
		wrap:   true,
	}
}

func (mdw *MarkdownWriter) WriteString(str string) (int, error) {
	var line string
	var rest string = str

	for len(rest) > 0 {
		var found bool
		line, rest, found = strings.Cut(rest, "\n")

		// A very crude implementation to avoid clobbering code
		if mdw.lineLen == 0 && strings.HasPrefix(line, "```") {
			mdw.wrap = !mdw.wrap
		}

		if !mdw.wrap {
			mdw.output.WriteString(line)
			mdw.output.WriteString("\n")
			continue
		}

		mdw.wrapLine(line)

		if found {
			mdw.output.WriteString("\n")
			mdw.lineLen = 0
		}
	}

	// TODO: Return errors
	return len(str), nil
}

func (mdw *MarkdownWriter) wrapLine(line string) {
	for len(line) > 0 {
		next := wordlikeRegexp.FindString(line)
		line = line[len(next):]

		if len(next) + mdw.lineLen > mdw.maxLen {
			next = strings.TrimSpace(next)

			if next != "" {
				mdw.output.WriteString("\n")
				mdw.lineLen = 0
			}
		}

		mdw.output.WriteString(next)
		mdw.lineLen += len(next)
	}
}
