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
	var readBuf = make([]byte, 4096)
	for {
		if r.discardingTooLongLine {
			eof, err := r.discardUntilNewline(file)
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

		// First, try to use any buffered overflow data
		if len(r.overflow) > 0 {
			n := len(r.overflow)
			if newlinePos := bytes.IndexByte(r.overflow, '\n'); newlinePos >= 0 {
				needed := newlinePos + 1
				if r.pos+needed > r.maxLineBytes {
					r.discardingTooLongLine = true
					fitBytes := r.maxLineBytes - r.pos
					if fitBytes > 0 {
						copy(r.buf[r.pos:], r.overflow[:fitBytes])
						r.pos += fitBytes
					}
					r.overflow = r.overflow[fitBytes:]
					truncated := string(stripWindowsLineEnding(r.buf[:r.pos]))
					return truncated, false, lineTooLongError{lineBytes: r.pos + len(r.overflow), maxLineBytes: r.maxLineBytes}
				}
				copy(r.buf[r.pos:], r.overflow[:needed])
				r.pos += needed
				r.overflow = r.overflow[needed:]
				continue
			}
			if r.pos+n > r.maxLineBytes {
				r.discardingTooLongLine = true
				fitBytes := r.maxLineBytes - r.pos
				if fitBytes > 0 {
					copy(r.buf[r.pos:], r.overflow[:fitBytes])
					r.pos += fitBytes
				}
				r.overflow = r.overflow[fitBytes:]
				// Return truncated line with error
				truncated := string(stripWindowsLineEnding(r.buf[:r.pos]))
				return truncated, false, lineTooLongError{lineBytes: r.pos + len(r.overflow), maxLineBytes: r.maxLineBytes}
			}
			copy(r.buf[r.pos:], r.overflow)
			r.pos += n
			r.overflow = nil
		} else {
			n, err := file.Read(readBuf)
			if n > 0 {
				if newlinePos := bytes.IndexByte(readBuf[:n], '\n'); newlinePos >= 0 {
					needed := newlinePos + 1
					if r.pos+needed > r.maxLineBytes {
						r.discardingTooLongLine = true
						fitBytes := r.maxLineBytes - r.pos
						if fitBytes > 0 {
							copy(r.buf[r.pos:], readBuf[:fitBytes])
							r.pos += fitBytes
						}
						r.overflow = append(r.overflow, readBuf[fitBytes:n]...)
						truncated := string(stripWindowsLineEnding(r.buf[:r.pos]))
						return truncated, false, lineTooLongError{lineBytes: r.pos + len(r.overflow), maxLineBytes: r.maxLineBytes}
					}
					copy(r.buf[r.pos:], readBuf[:needed])
					r.pos += needed
					if needed < n {
						r.overflow = append(r.overflow, readBuf[needed:n]...)
					}
					continue
				}
				if r.pos+n > r.maxLineBytes {
					r.discardingTooLongLine = true
					fitBytes := r.maxLineBytes - r.pos
					if fitBytes > 0 {
						copy(r.buf[r.pos:], readBuf[:fitBytes])
						r.pos += fitBytes
					}
					r.overflow = append(r.overflow, readBuf[fitBytes:n]...)
					// Return truncated line with error
					truncated := string(stripWindowsLineEnding(r.buf[:r.pos]))
					return truncated, false, lineTooLongError{lineBytes: r.pos + len(r.overflow), maxLineBytes: r.maxLineBytes}
				}
				copy(r.buf[r.pos:], readBuf[:n])
				r.pos += n
			}
			if err != nil {
				if err == io.EOF {
					return "", true, nil
				}
				return "", false, err
			}
		}
	}
}

func (r *lineReader) discardUntilNewline(file io.Reader) (bool, error) {
	readBuf := make([]byte, 4096)

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
				postNewline := r.overflow[newlineInOverflow+1:]
				r.pos = 0
				if len(postNewline) > 0 {
					copySize := len(postNewline)
					if copySize > r.maxLineBytes {
						copySize = r.maxLineBytes
					}
					copy(r.buf, postNewline[:copySize])
					r.pos = copySize
					if copySize < len(postNewline) {
						r.overflow = append(r.overflow[:0], postNewline[copySize:]...)
					} else {
						r.overflow = nil
					}
				} else {
					r.overflow = nil
				}
				r.discardingTooLongLine = false
				return false, nil
			}
			// No newline in overflow, need to read more
		}

		// Read more data
		n, err := file.Read(readBuf)
		if n > 0 {
			// Look for newline in new data
			newlineInRead := bytes.IndexByte(readBuf[:n], '\n')
			if newlineInRead >= 0 {
				// Found newline, keep all data after it split between buf and overflow.
				postNewline := readBuf[newlineInRead+1 : n]
				r.pos = 0
				if len(postNewline) > 0 {
					copySize := len(postNewline)
					if copySize > r.maxLineBytes {
						copySize = r.maxLineBytes
					}
					copy(r.buf, postNewline[:copySize])
					r.pos = copySize
					if copySize < len(postNewline) {
						r.overflow = append(r.overflow[:0], postNewline[copySize:]...)
					} else {
						r.overflow = nil
					}
				} else {
					r.overflow = nil
				}
				r.discardingTooLongLine = false
				return false, nil
			}
			// No newline yet, append to overflow and continue
			r.overflow = append(r.overflow, readBuf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}
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
}
