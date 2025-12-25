package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type collectingPrinter struct {
	lines []string
	opts  options
}

func (cp *collectingPrinter) printer(name string, lineNumber int, line string, prefixName bool) {
	var builder strings.Builder
	if prefixName {
		builder.WriteString(name)
		builder.WriteString(":")
	}
	if cp.opts.lineNumber {
		builder.WriteString(intToString(lineNumber))
		builder.WriteString(":")
	}
	builder.WriteString(line)
	cp.lines = append(cp.lines, builder.String())
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func TestGrepReaderMatches(t *testing.T) {
	input := strings.NewReader("foo\nbar\nbaz\n")
	cp := &collectingPrinter{opts: options{}}

	matched, err := grepReader(input, "stdin", func(line string) bool {
		return strings.Contains(line, "ba")
	}, cp.printer, cp.opts, false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matched {
		t.Fatalf("expected matched to be true")
	}

	want := []string{"bar", "baz"}
	if len(cp.lines) != len(want) {
		t.Fatalf("expected %d lines, got %d", len(want), len(cp.lines))
	}
	for i, line := range want {
		if cp.lines[i] != line {
			t.Fatalf("expected line %d to be %q, got %q", i, line, cp.lines[i])
		}
	}
}

func TestGrepReaderLineNumbersAndPrefix(t *testing.T) {
	input := strings.NewReader("alpha\nbeta\n")
	cp := &collectingPrinter{opts: options{lineNumber: true}}

	matched, err := grepReader(input, "file.txt", func(line string) bool {
		return true
	}, cp.printer, cp.opts, true)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matched {
		t.Fatalf("expected matched to be true")
	}

	want := []string{"file.txt:1:alpha", "file.txt:2:beta"}
	if len(cp.lines) != len(want) {
		t.Fatalf("expected %d lines, got %d", len(want), len(cp.lines))
	}
	for i, line := range want {
		if cp.lines[i] != line {
			t.Fatalf("expected line %d to be %q, got %q", i, line, cp.lines[i])
		}
	}
}

func TestGrepReaderLargeLine(t *testing.T) {
	largeLine := strings.Repeat("a", 200000)
	input := strings.NewReader(largeLine + "\n")
	cp := &collectingPrinter{opts: options{}}

	matched, err := grepReader(input, "stdin", func(line string) bool {
		return strings.HasPrefix(line, "aaa")
	}, cp.printer, cp.opts, false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matched {
		t.Fatalf("expected matched to be true")
	}
	if len(cp.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cp.lines))
	}
	if cp.lines[0] != largeLine {
		t.Fatalf("expected large line to match")
	}
}

func TestGrepReaderError(t *testing.T) {
	errBoom := errors.New("boom")
	reader := &failingReader{
		chunks: []string{"ok\n"},
		err:    errBoom,
	}
	cp := &collectingPrinter{opts: options{}}

	matched, err := grepReader(reader, "stdin", func(line string) bool {
		return strings.Contains(line, "ok")
	}, cp.printer, cp.opts, false)

	if !errors.Is(err, errBoom) {
		t.Fatalf("expected error %v, got %v", errBoom, err)
	}
	if !matched {
		t.Fatalf("expected matched to be true")
	}
	if len(cp.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cp.lines))
	}
}

func TestGrepFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("hello\nworld\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cp := &collectingPrinter{opts: options{}}
	matched, err := grepFile(path, func(line string) bool {
		return strings.Contains(line, "world")
	}, cp.printer, cp.opts, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matched {
		t.Fatalf("expected matched to be true")
	}
	if len(cp.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cp.lines))
	}
}

func TestGrepFileMissing(t *testing.T) {
	cp := &collectingPrinter{opts: options{}}
	matched, err := grepFile("does-not-exist.txt", func(line string) bool {
		return true
	}, cp.printer, cp.opts, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if matched {
		t.Fatalf("expected matched to be false")
	}
}

type failingReader struct {
	chunks []string
	err    error
	index  int
}

func (fr *failingReader) Read(p []byte) (int, error) {
	if fr.index < len(fr.chunks) {
		chunk := fr.chunks[fr.index]
		fr.index++
		return copy(p, chunk), nil
	}
	return 0, fr.err
}
