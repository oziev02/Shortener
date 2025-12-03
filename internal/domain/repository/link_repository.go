package repository

import (
	"context"

	"github.com/oziev02/Shortener/internal/domain/entity"
)

// LinkRepository определяет интерфейс для работы с ссылками
type LinkRepository interface {
	Create(ctx context.Context, link *entity.Link) error
	GetByShortURL(ctx context.Context, shortURL string) (*entity.Link, error)
	GetByCustomAlias(ctx context.Context, alias string) (*entity.Link, error)
	Exists(ctx context.Context, shortURL string) (bool, error)
}

// ClickRepository определяет интерфейс для работы с переходами
type ClickRepository interface {
	Create(ctx context.Context, click *entity.Click) error
	GetAnalytics(ctx context.Context, linkID int64) (*entity.Analytics, error)
	GetByLinkID(ctx context.Context, linkID int64, limit int) ([]*entity.Click, error)
}
