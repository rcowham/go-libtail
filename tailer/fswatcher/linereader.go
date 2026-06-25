// Copyright 2016-2018 The grok_exporter Authors
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
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const defaultMaxLineBytes = 1024 * 1024

type lineReader struct {
	buf                   []byte
	pos                   int
	maxLineBytes          int
	discardingTooLongLine bool
	overflow              []byte // data read but not yet used
	reader                *bufio.Reader
	source                io.Reader
}

type lineTooLongError struct {
	lineBytes    int
	maxLineBytes int
}

func (e lineTooLongError) Error() string {
	return fmt.Sprintf("line exceeds max length: %d > %d bytes", e.lineBytes, e.maxLineBytes)
}

func NewLineReader() *lineReader {
	return &lineReader{
		buf:          make([]byte, defaultMaxLineBytes),
		pos:          0,
		maxLineBytes: defaultMaxLineBytes,
	}
}

func NewLineReaderWithMaxLineBytes(maxLineBytes int) *lineReader {
	if maxLineBytes <= 0 {
		maxLineBytes = defaultMaxLineBytes
	}
	return &lineReader{
		buf:          make([]byte, maxLineBytes),
		pos:          0,
		maxLineBytes: maxLineBytes,
	}
}

// read the next line from the file.
// return values are (line, eof, err).
// * line is the line read.
// * eof is a boolean indicating if the end of file was reached before getting to the next '\n'.
// * err is set if an error other than io.EOF has occurred. err is never io.EOF.
// if eof is true, line is always "" and err always is nil.
// if eof is false and err is nil, an empty line means that there actually was an empty line in the file.
func (r *lineReader) ReadLine(file io.Reader) (string, bool, error) {
	if r.reader == nil || r.source != file {
		if r.reader != nil {
			// Preserve bytes already buffered in bufio.Reader before re-binding to a new source.
			if n := r.reader.Buffered(); n > 0 {
				if pending, err := r.reader.Peek(n); err == nil {
					copyPending := append([]byte(nil), pending...)
					r.overflow = append(r.overflow, copyPending...)
				}
			}
		}
		r.reader = bufio.NewReaderSize(file, 4096)
		r.source = file
	}

	for {
		if r.discardingTooLongLine {
			eof, err := r.discardUntilNewline()
			if err != nil {
				return "", false, err
			}
			if eof {
				return "", true, nil
			}
		}

		newlinePos := bytes.IndexByte(r.buf[:r.pos], '\n')
		if newlinePos >= 0 {
			result := make([]byte, newlinePos)
			copy(result, r.buf[:newlinePos])
			copy(r.buf, r.buf[newlinePos+1:r.pos])
			r.pos -= newlinePos + 1
			return string(stripWindowsLineEnding(result)), false, nil
		}

		chunk, err := r.readChunk()
		if len(chunk) > 0 {
			lineBytes := r.pos + len(chunk)
			if lineBytes > r.maxLineBytes {
				r.discardingTooLongLine = true
				fitBytes := r.maxLineBytes - r.pos
				if fitBytes > 0 {
					copy(r.buf[r.pos:], chunk[:fitBytes])
					r.pos += fitBytes
				}
				r.overflow = append(r.overflow[:0], chunk[fitBytes:]...)
				truncated := string(stripWindowsLineEnding(r.buf[:r.pos]))
				return truncated, false, lineTooLongError{lineBytes: lineBytes, maxLineBytes: r.maxLineBytes}
			}
			copy(r.buf[r.pos:], chunk)
			r.pos += len(chunk)
			continue
		}

		if err != nil {
			if err == io.EOF {
				return "", true, nil
			}
			return "", false, err
		}
	}
}

func (r *lineReader) discardUntilNewline() (bool, error) {
	for {
		// Look for newline in current buffer
		newlinePos := bytes.IndexByte(r.buf[:r.pos], '\n')
		if newlinePos >= 0 {
			// Found newline, shift buffer to keep only data after it
			copy(r.buf, r.buf[newlinePos+1:r.pos])
			r.pos -= newlinePos + 1
			r.discardingTooLongLine = false
			return false, nil
		}

		// Also check overflow buffer
		if len(r.overflow) > 0 {
			newlineInOverflow := bytes.IndexByte(r.overflow, '\n')
			if newlineInOverflow >= 0 {
				// Found newline in overflow, keep all data after it split between buf and overflow.
				r.setBufferFromBytes(r.overflow[newlineInOverflow+1:])
				r.discardingTooLongLine = false
				return false, nil
			}
			// No newline in overflow; discard all of it and continue scanning source.
			r.overflow = nil
		}

		chunk, err := r.readChunk()
		if len(chunk) > 0 {
			newlineInChunk := bytes.IndexByte(chunk, '\n')
			if newlineInChunk >= 0 {
				r.setBufferFromBytes(chunk[newlineInChunk+1:])
				r.discardingTooLongLine = false
				return false, nil
			}
		}

		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}
	}
}

func (r *lineReader) readChunk() ([]byte, error) {
	if len(r.overflow) > 0 {
		chunk := r.overflow
		r.overflow = nil
		return chunk, nil
	}

	chunk, err := r.reader.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return chunk, nil
	}
	return chunk, err
}

func (r *lineReader) setBufferFromBytes(data []byte) {
	r.pos = 0
	r.overflow = nil
	if len(data) == 0 {
		return
	}

	copySize := len(data)
	if copySize > r.maxLineBytes {
		copySize = r.maxLineBytes
	}
	copy(r.buf, data[:copySize])
	r.pos = copySize
	if copySize < len(data) {
		r.overflow = append(r.overflow[:0], data[copySize:]...)
	}
}

func stripWindowsLineEnding(s []byte) []byte {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	} else {
		return s
	}
}

func (r *lineReader) Clear() {
	r.pos = 0
	r.overflow = nil
	r.discardingTooLongLine = false
	r.reader = nil
	r.source = nil
}
