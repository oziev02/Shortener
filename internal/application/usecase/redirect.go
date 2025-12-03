package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/oziev02/Shortener/internal/domain/entity"
	"github.com/oziev02/Shortener/internal/domain/repository"
)

// RedirectUseCase обрабатывает редиректы по коротким ссылкам
type RedirectUseCase struct {
	linkRepo  repository.LinkRepository
	clickRepo repository.ClickRepository
	cache     Cache
}

// NewRedirectUseCase создаёт новый use case
func NewRedirectUseCase(
	linkRepo repository.LinkRepository,
	clickRepo repository.ClickRepository,
	cache Cache,
) *RedirectUseCase {
	return &RedirectUseCase{
		linkRepo:  linkRepo,
		clickRepo: clickRepo,
		cache:     cache,
	}
}

// Execute получает оригинальный URL и регистрирует переход
func (uc *RedirectUseCase) Execute(ctx context.Context, shortURL string, userAgent string, ipAddress string) (string, error) {
	var link *entity.Link
	var err error

	// Пытаемся получить из кэша
	if uc.cache != nil {
		cacheKey := fmt.Sprintf("link:%s", shortURL)
		cachedLink := &entity.Link{}
		if err := uc.cache.Get(ctx, cacheKey, cachedLink); err == nil {
			link = cachedLink
		}
	}

	// Если не в кэше, получаем из БД
	if link == nil {
		link, err = uc.linkRepo.GetByShortURL(ctx, shortURL)
		if err != nil {
			return "", fmt.Errorf("failed to get link: %w", err)
		}
		if link == nil {
			return "", ErrLinkNotFound
		}

		// Сохраняем в кэш
		if uc.cache != nil {
			cacheKey := fmt.Sprintf("link:%s", shortURL)
			if err := uc.cache.Set(ctx, cacheKey, link); err != nil {
				// Ошибка кэширования не критична, продолжаем работу
				_ = err
			}
		}
	}

	// Регистрируем переход
	click := &entity.Click{
		LinkID:    link.ID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ClickedAt: time.Now(),
	}
	if err := uc.clickRepo.Create(ctx, click); err != nil {
		// Логируем ошибку, но не прерываем редирект
		// В реальном приложении здесь должен быть логгер
		_ = err
	}

	return link.OriginalURL, nil
}
