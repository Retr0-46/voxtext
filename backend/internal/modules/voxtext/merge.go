package voxtext

import (
	"strings"
)

func Transcript(
	segments []SpeakerSegment,
	tokens []string,
	timestamps []float32,
	durations []float32,
) []DiarizedUtterance {
	count := min(len(tokens), len(timestamps))
	if count == 0 || len(segments) == 0 {
		return []DiarizedUtterance{}
	}

	result := make([]DiarizedUtterance, 0)
	current := DiarizedUtterance{}
	currentSegment := -1
	active := false

	appendCurrent := func() {
		current.Text = strings.TrimSpace(current.Text)
		if active && current.Text != "" {
			result = append(result, current)
		}
		current = DiarizedUtterance{}
		currentSegment = -1
		active = false
	}

	for index := 0; index < count; index++ {
		tokenStart := timestamps[index]
		tokenEnd := tokenEnd(index, count, timestamps, durations)
		segmentIndex := bestSegment(segments, tokenStart, tokenEnd)
		if segmentIndex == -1 {
			appendCurrent()
			continue
		}

		segment := segments[segmentIndex]
		if !active || current.Speaker != segment.Speaker || currentSegment != segmentIndex {
			appendCurrent()
			if strings.TrimSpace(tokens[index]) == "" {
				continue
			}

			current = DiarizedUtterance{
				Start:   max(tokenStart, segment.Start),
				End:     min(tokenEnd, segment.End),
				Speaker: segment.Speaker,
				Text:    tokens[index],
			}
			if current.End < current.Start {
				current.End = current.Start
			}
			currentSegment = segmentIndex
			active = true
			continue
		}

		current.Text += tokens[index]
		if end := min(tokenEnd, segment.End); end > current.End {
			current.End = end
		}
	}

	appendCurrent()
	return result
}

func tokenEnd(index, count int, timestamps, durations []float32) float32 {
	start := timestamps[index]
	if len(durations) == count && durations[index] > 0 {
		return start + durations[index]
	}
	if index+1 < count && timestamps[index+1] > start {
		return timestamps[index+1]
	}
	return start
}

func bestSegment(segments []SpeakerSegment, tokenStart, tokenEnd float32) int {
	bestIndex := -1
	bestOverlap := float32(-1)
	for index, segment := range segments {
		if tokenEnd == tokenStart {
			if tokenStart >= segment.Start && tokenStart <= segment.End && bestIndex == -1 {
				bestIndex = index
			}
			continue
		}

		overlap := min(tokenEnd, segment.End) - max(tokenStart, segment.Start)
		if overlap > bestOverlap && overlap > 0 {
			bestIndex = index
			bestOverlap = overlap
		}
	}
	return bestIndex
}
