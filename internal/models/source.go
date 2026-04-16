package models

import "time"

type Source struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	BaseURL               string     `json:"base_url"`
	Mode                  string     `json:"mode"`
	Active                bool       `json:"active"`
	ScrapeIntervalMinutes int        `json:"scrape_interval_minutes"`
	LastScrapedAt         *time.Time `json:"last_scraped_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
