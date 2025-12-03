package usecase

import (
	"context"
	"fmt"

	"github.com/oziev02/Shortener/internal/domain/entity"
	"github.com/oziev02/Shortener/internal/domain/repository"
)

// AnalyticsUseCase обрабатывает запросы аналитики
type AnalyticsUseCase struct {
	linkRepo  repository.LinkRepository
	clickRepo repository.ClickRepository
	cache     Cache
}

// NewAnalyticsUseCase создаёт новый use case
func NewAnalyticsUseCase(
	linkRepo repository.LinkRepository,
	clickRepo repository.ClickRepository,
	cache Cache,
) *AnalyticsUseCase {
	return &AnalyticsUseCase{
		linkRepo:  linkRepo,
		clickRepo: clickRepo,
		cache:     cache,
	}
}

// Execute получает аналитику по короткой ссылке
func (uc *AnalyticsUseCase) Execute(ctx context.Context, shortURL string) (*entity.Analytics, error) {
	// Получаем ссылку
	link, err := uc.linkRepo.GetByShortURL(ctx, shortURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}
	if link == nil {
		return nil, fmt.Errorf("link not found")
	}

	// Получаем аналитику
	analytics, err := uc.clickRepo.GetAnalytics(ctx, link.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	// Получаем последние переходы
	recentClicks, err := uc.clickRepo.GetByLinkID(ctx, link.ID, 10)
	if err == nil && recentClicks != nil {
		analytics.RecentClicks = make([]entity.Click, len(recentClicks))
		for i, click := range recentClicks {
			analytics.RecentClicks[i] = *click
		}
	}

	return analytics, nil
}
