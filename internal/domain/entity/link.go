package entity

import "time"

// Link представляет сокращённую ссылку
type Link struct {
	ID          int64     `json:"id"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CustomAlias string    `json:"custom_alias,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Click представляет переход по ссылке
type Click struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ClickedAt time.Time `json:"clicked_at"`
}

// Analytics представляет аналитику по ссылке
type Analytics struct {
	LinkID       int64            `json:"link_id"`
	ShortURL     string           `json:"short_url"`
	TotalClicks  int64            `json:"total_clicks"`
	ByDay        map[string]int64 `json:"by_day"`
	ByMonth      map[string]int64 `json:"by_month"`
	ByUserAgent  map[string]int64 `json:"by_user_agent"`
	RecentClicks []Click          `json:"recent_clicks,omitempty"`
}
