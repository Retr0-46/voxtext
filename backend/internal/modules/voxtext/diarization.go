package voxtext

import (
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

func NewDiarizer(config *Config) *sherpa.OfflineSpeakerDiarization {
	diarizerConfig := sherpa.OfflineSpeakerDiarizationConfig{}

	diarizerConfig.Segmentation.Pyannote.Model = config.SegmentationModel
	diarizerConfig.Segmentation.Pyannote.WindowShiftRatio = 0.1
	diarizerConfig.Segmentation.NumThreads = config.NumThreads
	diarizerConfig.Segmentation.Provider = "cpu"

	diarizerConfig.Embedding.Model = config.EmbeddingModel
	diarizerConfig.Embedding.NumThreads = config.NumThreads
	diarizerConfig.Embedding.Provider = "cpu"

	diarizerConfig.Clustering.NumClusters = 0
	diarizerConfig.Clustering.Threshold = config.DiarizationThreshold

	diarizerConfig.MinDurationOn = config.MinDurationOn
	diarizerConfig.MinDurationOff = config.MinDurationOff

	return sherpa.NewOfflineSpeakerDiarization(&diarizerConfig)
}
