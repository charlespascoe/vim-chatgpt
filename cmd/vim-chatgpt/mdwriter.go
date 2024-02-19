package main

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	bulletRegexp    = regexp.MustCompile(`^\s*([-*]|\d+\.) `)
	wordlikeRegexp  = regexp.MustCompile(`^(\s*)(\S+)\s`)
	quoteRegexp     = regexp.MustCompile(`^> `)
	leadingWsRegexp = regexp.MustCompile(`^\s+`)
	leadingNlRegexp = regexp.MustCompile(`^\n+`)
)

// var splitRe = regexp.MustCompile(`\s+`)

type MarkdownWriter struct {
	output   io.StringWriter
	maxLen   int
	tabWidth int
	line     strings.Builder
	wrap     bool
	buf      []byte
}

func NewMarkdownWriter(output io.StringWriter, maxLen, tabWidth int) *MarkdownWriter {
	return &MarkdownWriter{
		output:   output,
		maxLen:   maxLen,
		tabWidth: tabWidth,
		wrap:     true,
	}
}

func (mdw *MarkdownWriter) MaxLen() int {
	return mdw.maxLen
}

func (mdw *MarkdownWriter) TabWidth() int {
	return mdw.tabWidth
}

func (mdw *MarkdownWriter) WriteString(str string) (int, error) {
	// TODO: Properly handle write errors

	mdw.buf = append(mdw.buf, []byte(str)...)

	for {
		m := wordlikeRegexp.FindSubmatchIndex(mdw.buf)
		if m == nil {
			// No word match, but look for leading newlines and output them instead
			m = leadingNlRegexp.FindIndex(mdw.buf)

			if m != nil {
				mdw.buf = mdw.buf[m[1]:]
				mdw.output.WriteString(strings.Repeat("\n", m[1]))
				mdw.line.Reset()
			}

			return len(str), nil
		}

		ws := mdw.buf[m[2]:m[3]]
		word := mdw.buf[m[4]:m[5]]
		both := mdw.buf[m[2]:m[5]]

		mdw.buf = mdw.buf[m[5]:]

		mdw.writeWord(ws, word, both)

		if strings.HasPrefix(mdw.line.String(), "```") {
			mdw.wrap = !mdw.wrap
		}
	}
}

func (mdw *MarkdownWriter) writeWord(ws, word, both []byte) {
	output := both

	// TODO: Figure out how to nicely handle mixed whitespace and newlines in
	// non-wrap/"verbatim" mode
	nl := bytes.Count(ws, []byte("\n"))
	if nl > 0 {
		mdw.output.WriteString(strings.Repeat("\n", nl))
		mdw.line.Reset()
		output = word
	}

	if mdw.wrap && mdw.calcWidth(mdw.line.String(), string(output)) > mdw.maxLen {
		prefix := mdw.getPrefix()
		mdw.output.WriteString("\n")
		mdw.output.WriteString(prefix)
		mdw.line.Reset()
		mdw.line.WriteString(prefix)
		output = word
	}

	mdw.output.WriteString(string(output))
	mdw.line.Write(output)
}

func (mdw *MarkdownWriter) getPrefix() string {
	if str := bulletRegexp.FindString(mdw.line.String()); str != "" {
		return strings.Repeat(" ", len(str))
	} else if str := leadingWsRegexp.FindString(mdw.line.String()); str != "" {
		return str
	} else if quoteRegexp.MatchString(mdw.line.String()) {
		return "> "
	} else {
		return ""
	}
}

func (mdw *MarkdownWriter) calcWidth(strs ...string) int {
	// TODO: Perhaps instead use bytes and concatenate them to ensure proper
	// handling of multi-byte characters? For now assume that multi-byte
	// characters are never split across multiple strings
	var width int
	for _, str := range strs {
		width += utf8.RuneCountInString(str)
		// Account for tabs
		if mdw.tabWidth > 0 {
			width += strings.Count(str, "\t") * (mdw.tabWidth - 1)
		}
		// TODO: Perhaps handle different character widths?
	}
	return width
}

func (mdw *MarkdownWriter) Flush() error {
	if len(mdw.buf) == 0 {
		return nil
	}

	// TODO: Properly handle newlines in the buffer
	if mdw.line.Len()+len(mdw.buf) > mdw.maxLen {
		mdw.output.WriteString("\n")
		// Trim leading whitespace
		mdw.buf = leadingWsRegexp.ReplaceAll(mdw.buf, nil)
	}

	if _, err := mdw.output.WriteString(string(mdw.buf)); err != nil {
		return err
	}

	mdw.buf = mdw.buf[:0]

	return nil
}
