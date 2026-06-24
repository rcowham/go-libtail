// Copyright 2018-2020 The grok_exporter Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fswatcher

import (
	"strings"
	"testing"
)

func TestReadLineWithLongLineWithinLimit(t *testing.T) {
	r := NewLineReaderWithMaxLineBytes(1024 * 1024)
	input := strings.Repeat("a", 1024*1024-1) + "\n"

	line, eof, err := r.ReadLine(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadLine() returned unexpected error: %v", err)
	}
	if eof {
		t.Fatal("ReadLine() unexpectedly returned eof=true")
	}
	if len(line) != 1024*1024-1 {
		t.Fatalf("ReadLine() returned line length %d, expected %d", len(line), 1024*1024-1)
	}
}

func TestReadLineWithLongLineOverLimit(t *testing.T) {
	r := NewLineReaderWithMaxLineBytes(1024)
	input := strings.Repeat("b", 1025) + "\n"

	line, eof, err := r.ReadLine(strings.NewReader(input))
	if err == nil {
		t.Fatal("ReadLine() expected error for oversized line, got nil")
	}
	if _, ok := err.(lineTooLongError); !ok {
		t.Fatalf("ReadLine() returned wrong error type: %T", err)
	}
	if eof {
		t.Fatal("ReadLine() unexpectedly returned eof=true on oversized line")
	}
	// Check that truncated line is returned (1024 bytes)
	if len(line) != 1024 {
		t.Fatalf("ReadLine() returned truncated line length %d, expected %d", len(line), 1024)
	}
	if line != strings.Repeat("b", 1024) {
		t.Fatal("ReadLine() returned unexpected truncated line content")
	}
}

func TestReadLineSkipsOversizedLineOnNextCall(t *testing.T) {
	r := NewLineReaderWithMaxLineBytes(5)
	input := "123456\nok\n"

	// First call: truncated line with error
	line, eof, err := r.ReadLine(strings.NewReader(input))
	if err == nil {
		t.Fatal("ReadLine() expected error for oversized line, got nil")
	}
	if eof {
		t.Fatal("ReadLine() unexpectedly returned eof=true on oversized line")
	}
	// Check truncated line is returned (first 5 bytes: "12345")
	if line != "12345" {
		t.Fatalf("ReadLine() returned truncated line %q, expected %q", line, "12345")
	}

	// Second call: next complete line
	line, eof, err = r.ReadLine(strings.NewReader(""))
	if err != nil {
		t.Fatalf("second ReadLine() returned unexpected error: %v", err)
	}
	if eof {
		t.Fatal("second ReadLine() unexpectedly returned eof=true")
	}
	if line != "ok" {
		t.Fatalf("second ReadLine() returned %q, expected %q", line, "ok")
	}
}
