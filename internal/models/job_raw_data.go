package models

import "time"

type JobRawData struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	RawHTML   string    `json:"raw_html"`
	RawJSON   string    `json:"raw_json"`
	ScrapedAt time.Time `json:"scraped_at"`
}
