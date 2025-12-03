package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/oziev02/Shortener/internal/domain/entity"
	"github.com/oziev02/Shortener/internal/domain/repository"
)

// PostgresDB представляет подключение к PostgreSQL
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB создаёт новое подключение к PostgreSQL
func NewPostgresDB(dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	postgresDB := &PostgresDB{db: db}
	if err := postgresDB.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return postgresDB, nil
}

// migrate выполняет миграции базы данных
func (p *PostgresDB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS links (
			id SERIAL PRIMARY KEY,
			short_url VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			custom_alias VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_short_url ON links(short_url)`,
		`CREATE INDEX IF NOT EXISTS idx_links_custom_alias ON links(custom_alias)`,
		`CREATE TABLE IF NOT EXISTS clicks (
			id SERIAL PRIMARY KEY,
			link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
			user_agent TEXT,
			ip_address VARCHAR(45),
			clicked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clicks_link_id ON clicks(link_id)`,
		`CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at)`,
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// Close закрывает подключение к базе данных
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// LinkRepository реализует repository.LinkRepository
type LinkRepositoryImpl struct {
	db *PostgresDB
}

// NewLinkRepository создаёт новый репозиторий ссылок
func NewLinkRepository(db *PostgresDB) repository.LinkRepository {
	return &LinkRepositoryImpl{db: db}
}

func (r *LinkRepositoryImpl) Create(ctx context.Context, link *entity.Link) error {
	query := `INSERT INTO links (short_url, original_url, custom_alias, created_at) 
			  VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.db.db.QueryRowContext(ctx, query,
		link.ShortURL,
		link.OriginalURL,
		link.CustomAlias,
		link.CreatedAt,
	).Scan(&link.ID)

	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}

	return nil
}

func (r *LinkRepositoryImpl) GetByShortURL(ctx context.Context, shortURL string) (*entity.Link, error) {
	query := `SELECT id, short_url, original_url, custom_alias, created_at 
			  FROM links WHERE short_url = $1`

	link := &entity.Link{}
	err := r.db.db.QueryRowContext(ctx, query, shortURL).Scan(
		&link.ID,
		&link.ShortURL,
		&link.OriginalURL,
		&link.CustomAlias,
		&link.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}

	return link, nil
}

func (r *LinkRepositoryImpl) GetByCustomAlias(ctx context.Context, alias string) (*entity.Link, error) {
	query := `SELECT id, short_url, original_url, custom_alias, created_at 
			  FROM links WHERE custom_alias = $1`

	link := &entity.Link{}
	err := r.db.db.QueryRowContext(ctx, query, alias).Scan(
		&link.ID,
		&link.ShortURL,
		&link.OriginalURL,
		&link.CustomAlias,
		&link.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get link by alias: %w", err)
	}

	return link, nil
}

func (r *LinkRepositoryImpl) Exists(ctx context.Context, shortURL string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM links WHERE short_url = $1)`
	var exists bool
	err := r.db.db.QueryRowContext(ctx, query, shortURL).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

// ClickRepositoryImpl реализует repository.ClickRepository
type ClickRepositoryImpl struct {
	db *PostgresDB
}

// NewClickRepository создаёт новый репозиторий переходов
func NewClickRepository(db *PostgresDB) repository.ClickRepository {
	return &ClickRepositoryImpl{db: db}
}

func (r *ClickRepositoryImpl) Create(ctx context.Context, click *entity.Click) error {
	query := `INSERT INTO clicks (link_id, user_agent, ip_address, clicked_at) 
			  VALUES ($1, $2, $3, $4) RETURNING id`

	err := r.db.db.QueryRowContext(ctx, query,
		click.LinkID,
		click.UserAgent,
		click.IPAddress,
		click.ClickedAt,
	).Scan(&click.ID)

	if err != nil {
		return fmt.Errorf("failed to create click: %w", err)
	}

	return nil
}

func (r *ClickRepositoryImpl) GetAnalytics(ctx context.Context, linkID int64) (*entity.Analytics, error) {
	// Получаем информацию о ссылке
	linkQuery := `SELECT id, short_url FROM links WHERE id = $1`
	var linkIDFromDB int64
	var shortURL string
	err := r.db.db.QueryRowContext(ctx, linkQuery, linkID).Scan(&linkIDFromDB, &shortURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}

	analytics := &entity.Analytics{
		LinkID:      linkID,
		ShortURL:    shortURL,
		ByDay:       make(map[string]int64),
		ByMonth:     make(map[string]int64),
		ByUserAgent: make(map[string]int64),
	}

	// Общее количество переходов
	countQuery := `SELECT COUNT(*) FROM clicks WHERE link_id = $1`
	err = r.db.db.QueryRowContext(ctx, countQuery, linkID).Scan(&analytics.TotalClicks)
	if err != nil {
		return nil, fmt.Errorf("failed to get total clicks: %w", err)
	}

	// Группировка по дням
	dayQuery := `SELECT DATE(clicked_at) as day, COUNT(*) as count 
				 FROM clicks WHERE link_id = $1 
				 GROUP BY DATE(clicked_at) ORDER BY day DESC`
	rows, err := r.db.db.QueryContext(ctx, dayQuery, linkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks by day: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var day time.Time
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			continue
		}
		analytics.ByDay[day.Format("2006-01-02")] = count
	}

	// Группировка по месяцам
	monthQuery := `SELECT DATE_TRUNC('month', clicked_at)::date as month, COUNT(*) as count 
				   FROM clicks WHERE link_id = $1 
				   GROUP BY DATE_TRUNC('month', clicked_at) ORDER BY month DESC`
	rows, err = r.db.db.QueryContext(ctx, monthQuery, linkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks by month: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var month time.Time
		var count int64
		if err := rows.Scan(&month, &count); err != nil {
			continue
		}
		analytics.ByMonth[month.Format("2006-01")] = count
	}

	// Группировка по User-Agent
	uaQuery := `SELECT user_agent, COUNT(*) as count 
				FROM clicks WHERE link_id = $1 AND user_agent IS NOT NULL
				GROUP BY user_agent ORDER BY count DESC`
	rows, err = r.db.db.QueryContext(ctx, uaQuery, linkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks by user agent: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ua string
		var count int64
		if err := rows.Scan(&ua, &count); err != nil {
			continue
		}
		analytics.ByUserAgent[ua] = count
	}

	return analytics, nil
}

func (r *ClickRepositoryImpl) GetByLinkID(ctx context.Context, linkID int64, limit int) ([]*entity.Click, error) {
	query := `SELECT id, link_id, user_agent, ip_address, clicked_at 
			  FROM clicks WHERE link_id = $1 
			  ORDER BY clicked_at DESC LIMIT $2`

	rows, err := r.db.db.QueryContext(ctx, query, linkID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks: %w", err)
	}
	defer rows.Close()

	var clicks []*entity.Click
	for rows.Next() {
		click := &entity.Click{}
		if err := rows.Scan(
			&click.ID,
			&click.LinkID,
			&click.UserAgent,
			&click.IPAddress,
			&click.ClickedAt,
		); err != nil {
			continue
		}
		clicks = append(clicks, click)
	}

	return clicks, nil
}
