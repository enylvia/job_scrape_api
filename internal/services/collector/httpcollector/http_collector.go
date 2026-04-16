package httpcollector

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

type Collector struct {
	client *http.Client
}

type fetcher struct {
	client *http.Client
}

func New() *Collector {
	return NewWithTimeout(45 * time.Second)
}

func NewWithTimeout(timeout time.Duration) *Collector {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}

	return &Collector{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Collector) Mode() string {
	return collector.ModeHTTP
}

func (c *Collector) Collect(ctx context.Context, source models.Source, scraper collector.SourceScraper) ([]collector.CollectedJob, error) {
	return scraper.Collect(ctx, source, &fetcher{client: c.client})
}

func (f *fetcher) Fetch(ctx context.Context, url string) (collector.FetchResult, error) {
	startedAt := time.Now()
	log.Printf("%s %s %s", applogger.ColorScope("http"), applogger.ColorFetch("GET"), url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", "job-aggregator/phase2-http-collector")

	resp, err := f.client.Do(req)
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return collector.FetchResult{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	log.Printf("%s %s %s status=%d bytes=%d duration=%s", applogger.ColorScope("http"), applogger.ColorSuccess("OK"), url, resp.StatusCode, len(body), time.Since(startedAt))

	return collector.FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Body:       string(body),
		FetchedAt:  time.Now().UTC(),
	}, nil
}
