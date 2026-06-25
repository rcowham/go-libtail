package fswatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestReadNewLinesTruncatesAndContinuesOnLongLine(t *testing.T) {
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

	options := normalizeTailerOptions(TailerOptions{MaxLineBytes: 16})
	tailer := &fileTailer{
		options: options,
		lines:   make(chan *Line, 3),
		errors:  make(chan Error, 1),
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
		if !gotLine.Truncated {
			t.Fatal("expected truncated line to have Truncated=true")
		}
	default:
		t.Fatal("expected truncated line, got none")
	}

	// No long-line error should be emitted; truncation is indicated on the line itself.
	select {
	case gotErr := <-tailer.errors:
		t.Fatalf("unexpected error emitted: %v", gotErr)
	default:
	}

	// Finally expect the next complete line
	select {
	case gotLine := <-tailer.lines:
		if gotLine.Line != "short" {
			t.Fatalf("line=%q, want %q", gotLine.Line, "short")
		}
		if gotLine.Truncated {
			t.Fatal("expected normal line to have Truncated=false")
		}
	default:
		t.Fatal("expected a line after skipping oversized line, got none")
	}
}

func TestReadNewLinesNoTailErrorOnLongLine(t *testing.T) {
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
		lines:   make(chan *Line, 3),
		errors:  make(chan Error, 1),
		done:    make(chan struct{}),
	}

	fw := &fileWithReader{file: f, reader: NewLineReaderWithMaxLineBytes(options.MaxLineBytes)}
	if err := tailer.readNewLines(fw, logrus.New()); err != nil {
		t.Fatalf("readNewLines() returned unexpected error: %v", err)
	}

	select {
	case gotLine := <-tailer.lines:
		expectedTruncated := strings.Repeat("y", 16)
		if gotLine.Line != expectedTruncated {
			t.Fatalf("truncated line=%q, want %q", gotLine.Line, expectedTruncated)
		}
		if !gotLine.Truncated {
			t.Fatal("expected truncated line to have Truncated=true")
		}
	default:
		t.Fatal("expected truncated line, got none")
	}

	select {
	case gotErr := <-tailer.errors:
		t.Fatalf("unexpected error emitted: %v", gotErr)
	default:
	}

	select {
	case gotLine := <-tailer.lines:
		if gotLine.Line != "short" {
			t.Fatalf("line=%q, want %q", gotLine.Line, "short")
		}
		if gotLine.Truncated {
			t.Fatal("expected normal line to have Truncated=false")
		}
	default:
		t.Fatal("expected a line after skipping oversized line, got none")
	}
}
