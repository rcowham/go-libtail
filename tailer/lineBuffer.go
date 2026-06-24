// Copyright 2019-2020 The grok_exporter Authors
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

package tailer

import (
	"io"
	"sync"

	"github.com/rcowham/go-libtail/tailer/fswatcher"
)

// lineBuffer is a thread safe queue for *fswatcher.Line.
type lineBuffer interface {
	Push(line *fswatcher.Line)
	BlockingPop() *fswatcher.Line // can be interrupted by calling Close()
	Len() int
	io.Closer // will interrupt BlockingPop()
	Clear()
}

const lineBufferInitialSize = 1024

func NewLineBuffer() lineBuffer {
	return &lineBufferImpl{
		items:  make([]*fswatcher.Line, lineBufferInitialSize),
		lock:   sync.NewCond(&sync.Mutex{}),
		closed: false,
		empty:  true,
	}
}

type lineBufferImpl struct {
	items  []*fswatcher.Line
	head   int
	tail   int
	len    int
	lock   *sync.Cond
	closed bool
	empty  bool
}

func (b *lineBufferImpl) Push(line *fswatcher.Line) {
	b.lock.L.Lock()
	defer b.lock.L.Unlock()
	if b.closed {
		return
	}
	if b.len >= len(b.items) {
		b.grow()
	}
	wasEmpty := b.empty
	b.items[b.tail] = line
	b.tail = (b.tail + 1) % len(b.items)
	b.len++
	b.empty = false
	if wasEmpty {
		b.lock.Signal()
	}
}

func (b *lineBufferImpl) grow() {
	newSize := len(b.items) * 2
	newItems := make([]*fswatcher.Line, newSize)
	for i := 0; i < b.len; i++ {
		newItems[i] = b.items[(b.head+i)%len(b.items)]
	}
	b.items = newItems
	b.head = 0
	b.tail = b.len
}

// Interrupted by Close(), returns nil when Close() is called.
func (b *lineBufferImpl) BlockingPop() *fswatcher.Line {
	b.lock.L.Lock()
	defer b.lock.L.Unlock()
	for b.empty && !b.closed {
		b.lock.Wait()
	}
	if b.closed {
		return nil
	}
	line := b.items[b.head]
	b.items[b.head] = nil
	b.head = (b.head + 1) % len(b.items)
	b.len--
	if b.len == 0 {
		b.empty = true
	}
	return line
}

func (b *lineBufferImpl) Close() error {
	b.lock.L.Lock()
	defer b.lock.L.Unlock()
	if !b.closed {
		b.closed = true
		b.lock.Signal()
	}
	return nil
}

func (b *lineBufferImpl) Len() int {
	b.lock.L.Lock()
	defer b.lock.L.Unlock()
	return b.len
}

func (b *lineBufferImpl) Clear() {
	b.lock.L.Lock()
	defer b.lock.L.Unlock()
	for i := 0; i < b.len; i++ {
		b.items[(b.head+i)%len(b.items)] = nil
	}
	b.head = 0
	b.tail = 0
	b.len = 0
	b.empty = true
}
