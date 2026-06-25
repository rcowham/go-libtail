package fswatcher

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func benchmarkReadAllLines(b *testing.B, data string, maxLineBytes int, readBufferSize int) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		r := NewLineReaderWithOptions(maxLineBytes, readBufferSize)
		reader := strings.NewReader(data)
		for {
			_, eof, err := r.ReadLine(reader)
			if err != nil {
				b.Fatalf("ReadLine() error: %v", err)
			}
			if eof {
				break
			}
		}

		// Sanity-check that reader is consumed.
		if _, err := reader.Seek(0, io.SeekCurrent); err != nil {
			b.Fatalf("SeekCurrent() error: %v", err)
		}
	}
}

func BenchmarkLineReaderBufferSizes(b *testing.B) {
	shortLineData := strings.Repeat("short line\n", 20000)
	longLineData := strings.Repeat(strings.Repeat("x", 512)+"\n", 8000)

	bufferSizes := []int{4096, 32768, 65536}
	tests := []struct {
		name         string
		data         string
		maxLineBytes int
	}{
		{name: "short_lines", data: shortLineData, maxLineBytes: defaultMaxLineBytes},
		{name: "long_lines_512b", data: longLineData, maxLineBytes: defaultMaxLineBytes},
	}

	for _, tc := range tests {
		for _, size := range bufferSizes {
			name := fmt.Sprintf("%s_buf_%d", tc.name, size)
			b.Run(name, func(b *testing.B) {
				benchmarkReadAllLines(b, tc.data, tc.maxLineBytes, size)
			})
		}
	}
}
