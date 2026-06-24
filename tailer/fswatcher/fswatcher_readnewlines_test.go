package fswatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestReadNewLinesContinueOnLongLine(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	content := strings.Repeat("x", 32) + "\nshort\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer f.Close()

	options := normalizeTailerOptions(TailerOptions{MaxLineBytes: 16, ContinueOnLongLine: true})
	tailer := &fileTailer{
		options: options,
		lines:   make(chan *Line, 3),
		errors:  make(chan Error, 2),
		done:    make(chan struct{}),
	}

	fw := &fileWithReader{file: f, reader: NewLineReaderWithMaxLineBytes(options.MaxLineBytes)}
	if err := tailer.readNewLines(fw, logrus.New()); err != nil {
		t.Fatalf("readNewLines() returned unexpected error: %v", err)
	}

	// First, expect the truncated oversized line
	select {
	case gotLine := <-tailer.lines:
		expectedTruncated := strings.Repeat("x", 16)
		if gotLine.Line != expectedTruncated {
			t.Fatalf("truncated line=%q, want %q", gotLine.Line, expectedTruncated)
		}
	default:
		t.Fatal("expected truncated line, got none")
	}

	// Then expect the error
	select {
	case gotErr := <-tailer.errors:
		if gotErr.Type() != LineTooLong {
			t.Fatalf("error type=%v, want %v", gotErr.Type(), LineTooLong)
		}
	default:
		t.Fatal("expected TooLongLine error, got none")
	}

	// Finally expect the next complete line
	select {
	case gotLine := <-tailer.lines:
		if gotLine.Line != "short" {
			t.Fatalf("line=%q, want %q", gotLine.Line, "short")
		}
	default:
		t.Fatal("expected a line after skipping oversized line, got none")
	}
}

func TestReadNewLinesStopOnLongLineByDefault(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	content := strings.Repeat("y", 32) + "\nshort\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer f.Close()

	options := normalizeTailerOptions(TailerOptions{MaxLineBytes: 16})
	tailer := &fileTailer{
		options: options,
		lines:   make(chan *Line, 2),
		errors:  make(chan Error, 2),
		done:    make(chan struct{}),
	}

	fw := &fileWithReader{file: f, reader: NewLineReaderWithMaxLineBytes(options.MaxLineBytes)}
	tailErr := tailer.readNewLines(fw, logrus.New())
	if tailErr == nil {
		t.Fatal("readNewLines() expected error for oversized line, got nil")
	}
	if tailErr.Type() != LineTooLong {
		t.Fatalf("error type=%v, want %v", tailErr.Type(), LineTooLong)
	}
}
