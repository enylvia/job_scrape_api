package collector

import (
	"context"
	"time"

	"job_aggregator/internal/models"
)

const (
	ModeHTTP    = "http"
	ModeBrowser = "browser"
)

type FetchResult struct {
	URL        string
	StatusCode int
	Body       string
	FetchedAt  time.Time
}

type Fetcher interface {
	Fetch(ctx context.Context, url string) (FetchResult, error)
}

type Collector interface {
	Mode() string
	Collect(ctx context.Context, source models.Source, scraper SourceScraper) ([]CollectedJob, error)
}

type SourceScraper interface {
	Name() string
	Collect(ctx context.Context, source models.Source, fetcher Fetcher) ([]CollectedJob, error)
}

type CollectedJob struct {
	SourceJobURL           string
	SourceApplyURL         string
	Title                  string
	Slug                   string
	Company                string
	CompanyProfileImageURL string
	Location               string
	EmploymentType         string
	WorkplaceType          string
	Category               string
	SalaryMin              *int64
	SalaryMax              *int64
	Currency               string
	Description            string
	Requirements           string
	Benefits               string
	PostedAt               *time.Time
	ExpiredAt              *time.Time
	RawHTML                string
	RawJSON                string
	CollectedAt            time.Time
}
