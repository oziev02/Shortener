package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/oziev02/Shortener/internal/domain/entity"
	"github.com/oziev02/Shortener/internal/domain/repository"
	"github.com/oziev02/Shortener/internal/domain/service"
)

// ShortenUseCase обрабатывает создание коротких ссылок
type ShortenUseCase struct {
	linkRepo         repository.LinkRepository
	shortenerService *service.ShortenerService
	cache            Cache
}

// NewShortenUseCase создаёт новый use case
func NewShortenUseCase(
	linkRepo repository.LinkRepository,
	shortenerService *service.ShortenerService,
	cache Cache,
) *ShortenUseCase {
	return &ShortenUseCase{
		linkRepo:         linkRepo,
		shortenerService: shortenerService,
		cache:            cache,
	}
}

// CreateLinkRequest запрос на создание ссылки
type CreateLinkRequest struct {
	OriginalURL string `json:"original_url"`
	CustomAlias string `json:"custom_alias,omitempty"`
}

// CreateLinkResponse ответ с созданной ссылкой
type CreateLinkResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Execute создаёт новую короткую ссылку
func (uc *ShortenUseCase) Execute(ctx context.Context, req CreateLinkRequest) (*CreateLinkResponse, error) {
	var shortURL string
	var err error

	// Если указан кастомный алиас, используем его
	if req.CustomAlias != "" {
		// Проверяем, не занят ли алиас
		existing, err := uc.linkRepo.GetByCustomAlias(ctx, req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to check alias: %w", err)
		}
		if existing != nil {
			return nil, ErrAliasExists
		}
		shortURL = req.CustomAlias
	} else {
		// Генерируем случайный короткий URL
		for {
			shortURL, err = uc.shortenerService.GenerateShortURL(DefaultShortURLLength)
			if err != nil {
				return nil, fmt.Errorf("failed to generate short URL: %w", err)
			}

			// Проверяем уникальность
			exists, err := uc.linkRepo.Exists(ctx, shortURL)
			if err != nil {
				return nil, fmt.Errorf("failed to check uniqueness: %w", err)
			}
			if !exists {
				break
			}
		}
	}

	// Создаём ссылку
	link := &entity.Link{
		ShortURL:    shortURL,
		OriginalURL: req.OriginalURL,
		CustomAlias: req.CustomAlias,
		CreatedAt:   time.Now(),
	}

	if err := uc.linkRepo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	// Кэшируем ссылку
	if uc.cache != nil {
		cacheKey := fmt.Sprintf("link:%s", shortURL)
		if err := uc.cache.Set(ctx, cacheKey, link); err != nil {
			// Ошибка кэширования не критична, продолжаем работу
			_ = err
		}
	}

	return &CreateLinkResponse{
		ShortURL:    uc.shortenerService.BuildShortURL(shortURL),
		OriginalURL: req.OriginalURL,
	}, nil
}
