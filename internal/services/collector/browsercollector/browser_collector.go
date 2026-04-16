package browsercollector

import (
	"context"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"

	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

type Collector struct{}

type fetcher struct {
	browser playwright.Browser
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) Mode() string {
	return collector.ModeBrowser
}

func (c *Collector) Collect(ctx context.Context, source models.Source, scraper collector.SourceScraper) ([]collector.CollectedJob, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}
	defer func() {
		_ = pw.Stop()
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	defer func() {
		_ = browser.Close()
	}()

	return scraper.Collect(ctx, source, &fetcher{browser: browser})
}

func (f *fetcher) Fetch(ctx context.Context, url string) (collector.FetchResult, error) {
	page, err := f.browser.NewPage()
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("create page: %w", err)
	}
	defer func() {
		_ = page.Close()
	}()

	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(20000),
	})
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("navigate page: %w", err)
	}

	done := make(chan struct{})
	var html string
	var contentErr error

	go func() {
		defer close(done)
		html, contentErr = page.Content()
	}()

	select {
	case <-ctx.Done():
		return collector.FetchResult{}, ctx.Err()
	case <-done:
		if contentErr != nil {
			return collector.FetchResult{}, fmt.Errorf("get page content: %w", contentErr)
		}
	}

	return collector.FetchResult{
		URL:        url,
		StatusCode: 200,
		Body:       html,
		FetchedAt:  time.Now().UTC(),
	}, nil
}
