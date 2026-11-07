package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flamiry1/lvosa/backend/internal/models"
)

type GenerationRequestRepository interface {
	Create(ctx context.Context, req models.GenerationRequest) (int, error)
	Get(ctx context.Context, id, userID int) (models.GenerationRequest, error)
}

type GenerationService struct {
	generationsRepo GenerationRequestRepository
}

func NewGenerationService(g GenerationRequestRepository) *GenerationService {
	return &GenerationService{generationsRepo: g}

}

func checkTopic(topic string) (string, bool) {
	trimmedTopic := strings.TrimSpace(topic)
	if strings.Contains(trimmedTopic, "\r") {
		return "", false
	}
	if strings.Contains(trimmedTopic, "\n") {
		return "", false
	}
	lenTopic := utf8.RuneCountInString(trimmedTopic)
	if 1 <= lenTopic && lenTopic <= 80 {
		return trimmedTopic, true
	}
	return "", false
}

func (g *GenerationService) Create(ctx context.Context, req models.GenerationRequest) (int, error) {
	if !req.Level.IsValid() {
		return 0, models.ErrBadRequest
	}
	if !req.Difficulty.IsValid() {
		return 0, models.ErrBadRequest
	}
	if !req.Focus.IsValid() {
		return 0, models.ErrBadRequest
	}
	if !req.Size.IsValid() {
		return 0, models.ErrBadRequest
	}
	checkedTopic, isValid := checkTopic(req.Topic)
	if !isValid {
		return 0, models.ErrBadRequest
	}
	req.Topic = checkedTopic
	id, err := g.generationsRepo.Create(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("generation service create request: %w", err)
	}
	return id, nil
}

func (g *GenerationService) Get(ctx context.Context, id, userID int) (models.GenerationRequest, error) {
	req, err := g.generationsRepo.Get(ctx, id, userID)
	if err != nil {
		return models.GenerationRequest{}, fmt.Errorf("generation service get requst: %w", err)
	}
	return req, nil
}
