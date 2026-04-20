package models

import "time"

type Job struct {
	ID                     int64      `json:"id"`
	SourceID               int64      `json:"source_id"`
	SourceJobURL           string     `json:"source_job_url"`
	SourceApplyURL         string     `json:"source_apply_url"`
	Title                  string     `json:"title"`
	Slug                   string     `json:"slug"`
	Company                string     `json:"company"`
	CompanyProfileImageURL string     `json:"company_profile_image_url"`
	Location               string     `json:"location"`
	EmploymentType         string     `json:"employment_type"`
	WorkType               string     `json:"work_type"`
	Category               string     `json:"category"`
	SalaryMin              *int64     `json:"salary_min,omitempty"`
	SalaryMax              *int64     `json:"salary_max,omitempty"`
	Currency               string     `json:"currency"`
	Description            string     `json:"description"`
	Requirements           string     `json:"requirements"`
	Benefits               string     `json:"benefits"`
	PostedAt               *time.Time `json:"posted_at,omitempty"`
	ExpiredAt              *time.Time `json:"expired_at,omitempty"`
	ContentHash            string     `json:"content_hash"`
	Status                 string     `json:"status"`
	DuplicateOfJobID       *int64     `json:"duplicate_of_job_id,omitempty"`
	WordPressPostID        *int64     `json:"wordpress_post_id,omitempty"`
	TelegramSent           bool       `json:"telegram_sent"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}
