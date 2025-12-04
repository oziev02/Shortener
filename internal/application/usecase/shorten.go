package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
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
		// Проверяем, не занят ли алиас как custom_alias
		existing, err := uc.linkRepo.GetByCustomAlias(ctx, req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to check alias: %w", err)
		}
		if existing != nil {
			return nil, ErrAliasExists
		}

		// Проверяем, не занят ли алиас как short_url (так как shortURL = customAlias)
		exists, err := uc.linkRepo.Exists(ctx, req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to check short URL uniqueness: %w", err)
		}
		if exists {
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

			// Проверяем уникальность как short_url
			exists, err := uc.linkRepo.Exists(ctx, shortURL)
			if err != nil {
				return nil, fmt.Errorf("failed to check uniqueness: %w", err)
			}
			if exists {
				continue
			}

			// Также проверяем, не используется ли это значение как custom_alias
			existingByAlias, err := uc.linkRepo.GetByCustomAlias(ctx, shortURL)
			if err != nil {
				return nil, fmt.Errorf("failed to check alias uniqueness: %w", err)
			}
			if existingByAlias == nil {
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
		// Проверяем, не является ли это ошибкой уникальности PostgreSQL
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Ошибка уникальности - проверяем, какое ограничение нарушено
			constraintName := pqErr.Constraint

			// Если нарушено ограничение на custom_alias
			if constraintName == "links_custom_alias_key" {
				return nil, ErrAliasExists
			}

			// Если нарушено ограничение на short_url
			if constraintName == "links_short_url_key" {
				// Если был указан custom_alias, это тоже ошибка алиаса (так как shortURL = customAlias)
				if req.CustomAlias != "" {
					return nil, ErrAliasExists
				}
				// Если custom_alias не указан, это race condition - повторяем генерацию
				// Но это не должно происходить, так как мы проверяем перед созданием
				// Возвращаем общую ошибку для безопасности
				return nil, fmt.Errorf("short URL already exists, please try again: %w", err)
			}

			// Для других ограничений уникальности возвращаем общую ошибку
			return nil, ErrAliasExists
		}
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	// Кэшируем ссылку
	if uc.cache != nil {
		cacheKey := fmt.Sprintf("link:%s", shortURL)
		// Проверяем, что cache действительно не nil (для интерфейсов в Go)
		if cacheErr := uc.cache.Set(ctx, cacheKey, link); cacheErr != nil {
			// Ошибка кэширования не критична, продолжаем работу
			_ = cacheErr
		}
	}

	return &CreateLinkResponse{
		ShortURL:    uc.shortenerService.BuildShortURL(shortURL),
		OriginalURL: req.OriginalURL,
	}, nil
}
