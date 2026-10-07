package voxtext

import "errors"

var (
	ErrNoModelsPath         = errors.New("models directory is not configured")
	ErrNoFFmpegPath         = errors.New("ffmpeg executable is not configured")
	ErrUnexpectedValueInEnv = errors.New("invalid configuration value")
	ErrServiceClosed        = errors.New("transcription service is closed")
)

type DiarizedUtterance struct {
	Start   float32 `json:"start"`
	End     float32 `json:"end"`
	Speaker int     `json:"speaker"`
	Text    string  `json:"text"`
}

type SpeakerSegment struct {
	Start   float32
	End     float32
	Speaker int
}
