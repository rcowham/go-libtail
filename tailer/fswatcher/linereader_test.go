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
	"io"
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

func TestReadLineRespectsNewlineBeforeMaxAcrossLargeRead(t *testing.T) {
	r := NewLineReaderWithMaxLineBytes(100)
	input := strings.Join([]string{
		"Perforce server info:",
		"\t2020/01/11 02:00:02 pid 25396 p4sdp@p4poke-chi 127.0.0.1 [p4/2019.2.PREP-TEST_ONLY/LINUX26X86_64/1891638] 'user-serverid'",
		"Perforce server info:",
		"\t2020/01/11 02:00:02 pid 25390 completed .008s 0+0us 0+8io 0+0net 7632k 0pf",
	}, "\n") + "\n"
	reader := strings.NewReader(input)

	line, eof, err := r.ReadLine(reader)
	if err != nil {
		t.Fatalf("first ReadLine() returned unexpected error: %v", err)
	}
	if eof {
		t.Fatal("first ReadLine() unexpectedly returned eof=true")
	}
	if line != "Perforce server info:" {
		t.Fatalf("first ReadLine() returned %q", line)
	}

	line, eof, err = r.ReadLine(reader)
	if err == nil {
		t.Fatal("second ReadLine() expected lineTooLongError, got nil")
	}
	if _, ok := err.(lineTooLongError); !ok {
		t.Fatalf("second ReadLine() returned wrong error type: %T", err)
	}
	if eof {
		t.Fatal("second ReadLine() unexpectedly returned eof=true")
	}
	if !strings.HasPrefix(line, "\t2020/01/11 02:00:02 pid 25396") {
		t.Fatalf("second ReadLine() returned unexpected truncated content: %q", line)
	}

	line, eof, err = r.ReadLine(reader)
	if err != nil {
		t.Fatalf("third ReadLine() returned unexpected error: %v", err)
	}
	if eof {
		t.Fatal("third ReadLine() unexpectedly returned eof=true")
	}
	if line != "Perforce server info:" {
		t.Fatalf("third ReadLine() returned %q", line)
	}

	line, eof, err = r.ReadLine(reader)
	if err != nil {
		t.Fatalf("fourth ReadLine() returned unexpected error: %v", err)
	}
	if eof {
		t.Fatal("fourth ReadLine() unexpectedly returned eof=true")
	}
	if line != "\t2020/01/11 02:00:02 pid 25390 completed .008s 0+0us 0+8io 0+0net 7632k 0pf" {
		t.Fatalf("fourth ReadLine() returned %q", line)
	}

	line, eof, err = r.ReadLine(reader)
	if err != nil {
		t.Fatalf("final ReadLine() returned unexpected error: %v", err)
	}
	if !eof {
		t.Fatalf("final ReadLine() eof=false, line=%q", line)
	}

	_, _, err = r.ReadLine(reader)
	if err != nil && err != io.EOF {
		t.Fatalf("extra ReadLine() returned unexpected error: %v", err)
	}
}
