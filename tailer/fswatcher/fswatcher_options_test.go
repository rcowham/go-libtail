package fswatcher

import "testing"

func TestNormalizeTailerOptionsUsesDefaultMaxLineBytes(t *testing.T) {
	options := normalizeTailerOptions(TailerOptions{})
	if options.MaxLineBytes != defaultMaxLineBytes {
		t.Fatalf("MaxLineBytes=%d, want %d", options.MaxLineBytes, defaultMaxLineBytes)
	}
}

func TestNormalizeTailerOptionsKeepsCustomMaxLineBytes(t *testing.T) {
	options := normalizeTailerOptions(TailerOptions{MaxLineBytes: 2048})
	if options.MaxLineBytes != 2048 {
		t.Fatalf("MaxLineBytes=%d, want %d", options.MaxLineBytes, 2048)
	}
}

func TestNormalizeTailerOptionsKeepsContinueOnLongLine(t *testing.T) {
	options := normalizeTailerOptions(TailerOptions{ContinueOnLongLine: true})
	if !options.ContinueOnLongLine {
		t.Fatal("ContinueOnLongLine=false, want true")
	}
}
