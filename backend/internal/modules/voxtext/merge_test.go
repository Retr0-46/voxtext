package voxtext

import (
	"testing"
)

func TestTranscriptSplitsSegmentsAndSpeakers(t *testing.T) {
	segments := []SpeakerSegment{
		{Start: 0, End: 1, Speaker: 0},
		{Start: 1.2, End: 2, Speaker: 0},
		{Start: 2, End: 3, Speaker: 1},
	}
	tokens := []string{"hello", " ", "again", "bye"}
	timestamps := []float32{0.1, 0.5, 1.3, 2.1}
	durations := []float32{0.2, 0.1, 0.2, 0.3}

	result := Transcript(segments, tokens, timestamps, durations)

	if len(result) != 3 {
		t.Fatalf("utterance count: got %d, want 3: %+v", len(result), result)
	}
	if result[0].Speaker != 0 || result[0].Text != "hello" {
		t.Fatalf("first utterance: %+v", result[0])
	}
	if result[1].Speaker != 0 || result[1].Text != "again" {
		t.Fatalf("second utterance: %+v", result[1])
	}
	if result[2].Speaker != 1 || result[2].Text != "bye" {
		t.Fatalf("third utterance: %+v", result[2])
	}
}

func TestTranscriptReturnsNonNilEmptySlice(t *testing.T) {
	result := Transcript(nil, []string{"hello"}, []float32{0.1}, nil)
	if result == nil || len(result) != 0 {
		t.Fatalf("got %#v, want a non-nil empty slice", result)
	}
}
