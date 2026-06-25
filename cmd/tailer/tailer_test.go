// Copyright 2019 Robert Cowham, Perforce Software
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

package main

import "testing"

func TestFormatOutputLineDecodeEscapes(t *testing.T) {
	in := `Perforce server info:\n\t2020/01/11 02:00:02 pid 253`
	got := formatOutputLine(in, true)
	want := "Perforce server info:\n\t2020/01/11 02:00:02 pid 253"
	if got != want {
		t.Fatalf("formatOutputLine()=%q, want %q", got, want)
	}
}

func TestFormatOutputLineDecodeDisabled(t *testing.T) {
	in := `Perforce server info:\n\t2020/01/11 02:00:02 pid 253`
	got := formatOutputLine(in, false)
	if got != in {
		t.Fatalf("formatOutputLine()=%q, want %q", got, in)
	}
}
