package voxtext

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultMaxUploadBytes = int64(500 << 20)

type Config struct {
	SegmentationModel string
	EmbeddingModel    string
	ASRModel          string
	Tokens            string
	FFmpegPath        string
	UploadDir         string
	NumThreads        int

	DiarizationThreshold float32
	MinDurationOn        float32
	MinDurationOff       float32
	MaxUploadBytes       int64
}

// LoadConfig builds transcription settings from the process environment.
// The application loads .env before calling this function.
func LoadConfig() (Config, error) {
	modelsDir := strings.TrimSpace(os.Getenv("MODELS_DIR"))
	if modelsDir == "" {
		modelsDir = filepath.Join("internal", "modules", "voxtext", "models")
	}
	modelsDir, err := filepath.Abs(modelsDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve models directory: %w", err)
	}

	ffmpegPath := strings.TrimSpace(os.Getenv("FFMPEG"))
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	numThreads, err := positiveInt("NUM_THREADS", 4)
	if err != nil {
		return Config{}, err
	}
	threshold, err := float32Value("DIARIZATION_THRESHOLD", 0.8, 0, 1)
	if err != nil {
		return Config{}, err
	}
	minOn, err := float32Value("MIN_DURATION_ON", 0.5, 0, math.MaxFloat32)
	if err != nil {
		return Config{}, err
	}
	minOff, err := float32Value("MIN_DURATION_OFF", 0.5, 0, math.MaxFloat32)
	if err != nil {
		return Config{}, err
	}
	maxUploadBytes, err := positiveInt64("MAX_UPLOAD_BYTES", defaultMaxUploadBytes)
	if err != nil {
		return Config{}, err
	}

	uploadDir := strings.TrimSpace(os.Getenv("UPLOAD_TMP_DIR"))
	if uploadDir == "" {
		uploadDir = os.TempDir()
	}
	uploadDir, err = filepath.Abs(uploadDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve upload directory: %w", err)
	}

	cfg := Config{
		SegmentationModel:    filepath.Join(modelsDir, "diarization", "pyannote", "model.int8.onnx"),
		EmbeddingModel:       filepath.Join(modelsDir, "embeddings", "titanet", "model.onnx"),
		ASRModel:             filepath.Join(modelsDir, "asr", "gigaam-ctc", "model.int8.onnx"),
		Tokens:               filepath.Join(modelsDir, "asr", "gigaam-ctc", "tokens.txt"),
		FFmpegPath:           ffmpegPath,
		UploadDir:            uploadDir,
		NumThreads:           numThreads,
		DiarizationThreshold: threshold,
		MinDurationOn:        minOn,
		MinDurationOff:       minOff,
		MaxUploadBytes:       maxUploadBytes,
	}

	for label, path := range map[string]string{
		"segmentation model": cfg.SegmentationModel,
		"embedding model":    cfg.EmbeddingModel,
		"ASR model":          cfg.ASRModel,
		"tokens":             cfg.Tokens,
	} {
		info, err := os.Stat(path)
		if err != nil {
			return Config{}, fmt.Errorf("%s %q: %w", label, path, err)
		}
		if !info.Mode().IsRegular() {
			return Config{}, fmt.Errorf("%s %q is not a regular file", label, path)
		}
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o700); err != nil {
		return Config{}, fmt.Errorf("create upload directory %q: %w", cfg.UploadDir, err)
	}
	return cfg, nil
}

func positiveInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return number, nil
}

func positiveInt64(name string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return number, nil
}

func float32Value(name string, fallback, minimum, maximum float64) (float32, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return float32(fallback), nil
	}
	number, err := strconv.ParseFloat(value, 32)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < minimum || number > maximum {
		return 0, fmt.Errorf("%s must be between %g and %g", name, minimum, maximum)
	}
	return float32(number), nil
}
