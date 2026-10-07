package voxtext

import (
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

func NewRecognizer(config *Config) *sherpa.OfflineRecognizer {
	recognizerConfig := sherpa.OfflineRecognizerConfig{}

	recognizerConfig.FeatConfig.SampleRate = 16000
	recognizerConfig.FeatConfig.FeatureDim = 80

	recognizerConfig.ModelConfig.NemoCTC.Model = config.ASRModel
	recognizerConfig.ModelConfig.Tokens = config.Tokens

	recognizerConfig.ModelConfig.Provider = "cpu"
	recognizerConfig.ModelConfig.NumThreads = config.NumThreads

	recognizerConfig.DecodingMethod = "greedy_search"

	return sherpa.NewOfflineRecognizer(&recognizerConfig)
}
