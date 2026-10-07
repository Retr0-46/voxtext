package voxtext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

type Service struct {
	recognizer *sherpa.OfflineRecognizer
	diarizer   *sherpa.OfflineSpeakerDiarization
	converter  *Converter

	gate    chan struct{}
	stateMu sync.Mutex
	closed  bool
}

func NewService(cfg *Config, converter *Converter) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if converter == nil {
		return nil, errors.New("audio converter is nil")
	}

	recognizer := NewRecognizer(cfg)
	if recognizer == nil {
		return nil, errors.New("create offline recognizer")
	}
	diarizer := NewDiarizer(cfg)
	if diarizer == nil {
		sherpa.DeleteOfflineRecognizer(recognizer)
		return nil, errors.New("create offline speaker diarizer")
	}

	return &Service{
		recognizer: recognizer,
		diarizer:   diarizer,
		converter:  converter,
		gate:       make(chan struct{}, 1),
	}, nil
}

// Close waits for an active transcription, releases the native Sherpa objects,
// and prevents new work from starting. It is safe to call more than once.
func (s *Service) Close() {
	if s == nil {
		return
	}

	s.stateMu.Lock()
	if s.closed {
		s.stateMu.Unlock()
		return
	}
	s.closed = true
	gate := s.gate
	s.stateMu.Unlock()
	if gate == nil {
		return
	}

	gate <- struct{}{}
	defer func() { <-gate }()

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.diarizer != nil {
		sherpa.DeleteOfflineSpeakerDiarization(s.diarizer)
		s.diarizer = nil
	}
	if s.recognizer != nil {
		sherpa.DeleteOfflineRecognizer(s.recognizer)
		s.recognizer = nil
	}
}

func (s *Service) Transcribe(ctx context.Context, inputPath string) ([]DiarizedUtterance, error) {
	if s == nil {
		return nil, errors.New("transcription service is nil")
	}
	if ctx == nil {
		return nil, errors.New("transcription context is nil")
	}
	if strings.TrimSpace(inputPath) == "" {
		return nil, errors.New("input audio path is empty")
	}
	if s.gate == nil {
		return nil, ErrServiceClosed
	}

	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	s.stateMu.Lock()
	if s.closed || s.recognizer == nil || s.diarizer == nil || s.converter == nil {
		s.stateMu.Unlock()
		return nil, ErrServiceClosed
	}
	recognizer := s.recognizer
	diarizer := s.diarizer
	converter := s.converter
	s.stateMu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	wavPath, err := converter.Convert(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("convert audio: %w", err)
	}
	defer os.Remove(wavPath)

	wave := sherpa.ReadWave(wavPath)
	if wave == nil || wave.SampleRate <= 0 || len(wave.Samples) == 0 {
		return nil, errors.New("read converted WAV audio")
	}
	if expectedSampleRate := diarizer.SampleRate(); expectedSampleRate > 0 && wave.SampleRate != expectedSampleRate {
		return nil, fmt.Errorf("unexpected sample rate: got %d Hz, want %d Hz", wave.SampleRate, expectedSampleRate)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	diarizationSegments := diarizer.Process(wave.Samples)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(diarizationSegments) == 0 {
		return []DiarizedUtterance{}, nil
	}

	stream := sherpa.NewOfflineStream(recognizer)
	if stream == nil {
		return nil, errors.New("create offline recognition stream")
	}
	defer sherpa.DeleteOfflineStream(stream)

	stream.AcceptWaveform(wave.SampleRate, wave.Samples)
	recognizer.Decode(stream)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := stream.GetResult()
	if result == nil {
		return nil, errors.New("get offline recognition result")
	}
	if len(result.Tokens) == 0 {
		return []DiarizedUtterance{}, nil
	}
	if len(result.Tokens) != len(result.Timestamps) {
		return nil, fmt.Errorf("recognizer returned %d tokens and %d timestamps", len(result.Tokens), len(result.Timestamps))
	}

	segments := make([]SpeakerSegment, len(diarizationSegments))
	for index, segment := range diarizationSegments {
		segments[index] = SpeakerSegment{
			Start: segment.Start, End: segment.End, Speaker: segment.Speaker,
		}
	}

	return Transcript(segments, result.Tokens, result.Timestamps, result.Durations), nil
}
