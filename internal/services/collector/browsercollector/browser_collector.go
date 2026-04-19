package browsercollector

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/playwright-community/playwright-go"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

type Collector struct{}

type fetcher struct {
	browser  playwright.Browser
	source   models.Source
	headless bool
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

	headless := resolveHeadlessMode(source)
	log.Printf("%s %s source=%s headless=%t", applogger.ColorScope("browser"), applogger.ColorStart("LAUNCH"), source.Name, headless)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	defer func() {
		_ = browser.Close()
	}()

	return scraper.Collect(ctx, source, &fetcher{
		browser:  browser,
		source:   source,
		headless: headless,
	})
}

func (f *fetcher) Fetch(ctx context.Context, url string) (collector.FetchResult, error) {
	contextOptions := playwright.BrowserNewContextOptions{
		UserAgent:  playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"),
		Locale:     playwright.String("en-US"),
		TimezoneId: playwright.String("Asia/Jakarta"),
		Viewport: &playwright.Size{
			Width:  1440,
			Height: 960,
		},
	}

	browserContext, err := f.browser.NewContext(contextOptions)
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("create browser context: %w", err)
	}
	defer func() {
		_ = browserContext.Close()
	}()

	if err := browserContext.AddInitScript(playwright.Script{
		Content: playwright.String(`
Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });
Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3] });
window.chrome = window.chrome || { runtime: {} };
		`),
	}); err != nil {
		return collector.FetchResult{}, fmt.Errorf("add init script: %w", err)
	}

	page, err := browserContext.NewPage()
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("create page: %w", err)
	}
	defer func() {
		_ = page.Close()
	}()

	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(resolveNavigationTimeoutMillis(ctx, 30000)),
	})
	if err != nil {
		return collector.FetchResult{}, fmt.Errorf("navigate page: %w", err)
	}

	if f.source.Name == "glints" {
		_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State:   playwright.LoadStateLoad,
			Timeout: playwright.Float(resolveNavigationTimeoutMillis(ctx, 15000)),
		})
		page.WaitForTimeout(3000)
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

func resolveHeadlessMode(source models.Source) bool {
	if value, ok := parseHeadlessEnv("BROWSER_COLLECTOR_HEADLESS"); ok {
		return value
	}

	if source.Name == "glints" {
		if value, ok := parseHeadlessEnv("GLINTS_BROWSER_HEADLESS"); ok {
			return value
		}

		return true
	}

	return true
}

func parseHeadlessEnv(key string) (bool, bool) {
	value := os.Getenv(key)
	if value == "" {
		return false, false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}

	return parsed, true
}

func resolveNavigationTimeoutMillis(ctx context.Context, fallback float64) float64 {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 1
		}

		timeoutMillis := float64(remaining.Milliseconds())
		if timeoutMillis < 1 {
			return 1
		}

		if timeoutMillis < fallback {
			return timeoutMillis
		}
	}

	return fallback
}
