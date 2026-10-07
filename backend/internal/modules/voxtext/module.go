package voxtext

import "fmt"

type Module struct {
	Handler *Handler
	service *Service
}

func NewModule(cfg Config) (*Module, error) {
	converter, err := NewConverter(cfg.FFmpegPath)
	if err != nil {
		return nil, fmt.Errorf("create audio converter: %w", err)
	}
	service, err := NewService(&cfg, converter)
	if err != nil {
		return nil, fmt.Errorf("create transcription service: %w", err)
	}
	return &Module{
		Handler: NewHandler(service, cfg.MaxUploadBytes, cfg.UploadDir),
		service: service,
	}, nil
}

func (m *Module) Close() {
	if m != nil {
		m.service.Close()
	}
}
