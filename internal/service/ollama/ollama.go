package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	cfg "github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/model"
	"log/slog"
	"net/http"
	"time"
)

type Service struct {
	client  *http.Client
	baseURL string
	logger  *slog.Logger
}

func NewOllamaService(cfg cfg.AppConfig, logger *slog.Logger) *Service {
	return &Service{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		baseURL: fmt.Sprintf(
			"http://%s:%s",
			cfg.OllamaHost,
			cfg.OllamaPort,
		),
		logger: logger,
	}
}

func (s *Service) Generate(
	ctx context.Context,
	reqBody model.GenerateRequest,
) (*model.GenerateResponse, error) {

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/generate",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out model.GenerateResponse

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (s *Service) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+"/api/tags",
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *Service) ListModels(
	ctx context.Context,
) (*model.TagsResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+"/api/tags",
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out model.TagsResponse

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
