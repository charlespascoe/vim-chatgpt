package main

import (
	"io"
	"strings"
)

type ReplaceWriter struct {
	output io.StringWriter
	from, to string
}

func NewReplaceWriter(output io.StringWriter, from, to string) *ReplaceWriter {
	return &ReplaceWriter{output, from, to}
}

func (rw *ReplaceWriter) WriteString(str string) (int, error) {
	return rw.output.WriteString(strings.ReplaceAll(str, rw.from, rw.to))
}
