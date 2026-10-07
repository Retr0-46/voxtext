package voxtext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const tempFilePattern = "voxtext-*.wav"

type Converter struct {
	ffmpegPath string
}

func NewConverter(ffmpegPath string) (*Converter, error) {
	if strings.TrimSpace(ffmpegPath) == "" {
		return nil, errors.New("ffmpeg path is empty")
	}

	resolvedPath, err := exec.LookPath(ffmpegPath)
	if err != nil {
		return nil, fmt.Errorf("find ffmpeg executable %q: %w", ffmpegPath, err)
	}

	return &Converter{ffmpegPath: resolvedPath}, nil
}

// Convert converts inputPath to a mono 16 kHz PCM WAV file.
// The caller owns the returned file and must remove it after use.
func (c *Converter) Convert(ctx context.Context, inputPath string) (string, error) {
	if c == nil {
		return "", errors.New("audio converter is nil")
	}
	if ctx == nil {
		return "", errors.New("audio conversion context is nil")
	}
	if strings.TrimSpace(inputPath) == "" {
		return "", errors.New("input audio path is empty")
	}

	tempFile, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		return "", fmt.Errorf("create temporary WAV file: %w", err)
	}

	tempPath := tempFile.Name()
	fileClosed := false
	conversionSucceeded := false

	defer func() {
		if !fileClosed {
			_ = tempFile.Close()
		}
		if !conversionSucceeded {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("close temporary WAV file: %w", err)
	}
	fileClosed = true

	cmd := exec.CommandContext(
		ctx,
		c.ffmpegPath,
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-c:a", "pcm_s16le",
		tempPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("convert audio: %w", ctxErr)
		}

		message := strings.TrimSpace(string(output))
		if message != "" {
			return "", fmt.Errorf("ffmpeg conversion failed: %w: %s", err, message)
		}
		return "", fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	conversionSucceeded = true
	return tempPath, nil
}
