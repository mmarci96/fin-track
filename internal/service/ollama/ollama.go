package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	cfg "github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/model"
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
		return nil, errors.Wrap(err, "marshal generate request")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/generate",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, errors.Wrap(err, "build generate request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "call ollama generate")
	}
	defer resp.Body.Close()

	var out model.GenerateResponse

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errors.Wrap(err, "decode generate response")
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
		return errors.Wrap(err, "build healthcheck request")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "call ollama healthcheck")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Newf("ollama healthcheck returned status %d", resp.StatusCode)
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
		return nil, errors.Wrap(err, "build list models request")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "call ollama list models")
	}
	defer resp.Body.Close()

	var out model.TagsResponse

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errors.Wrap(err, "decode list models response")
	}

	return &out, nil
}
