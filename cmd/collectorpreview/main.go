package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
	"job_aggregator/internal/services/collector/browsercollector"
	"job_aggregator/internal/services/collector/httpcollector"
	"job_aggregator/internal/services/collector/sources"
)

func main() {
	pageLimit := getEnvIntWithLegacy("COLLECTOR_PREVIEW_PAGES", "DEALLS_PREVIEW_PAGES", 1)
	printLimit := getEnvIntWithLegacy("COLLECTOR_PREVIEW_PRINT_LIMIT", "DEALLS_PREVIEW_PRINT_LIMIT", 3)
	sourceName := getEnvString("COLLECTOR_PREVIEW_SOURCE", "dealls")

	log.Printf("%s %s collector preview source=%s (pages=%d print_limit=%d)", applogger.ColorScope("preview"), applogger.ColorStart("START"), sourceName, pageLimit, printLimit)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	source, engine, scraper := buildPreviewConfig(sourceName, pageLimit)

	log.Printf("%s %s jobs from configured preview source", applogger.ColorScope("preview"), applogger.ColorFetch("COLLECT"))
	jobs, err := engine.Collect(ctx, source, scraper)
	if err != nil {
		log.Fatalf("%s %s preview scrape failed: %v", applogger.ColorScope("preview"), applogger.ColorError("ERROR"), err)
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs collected.")
		return
	}

	if printLimit > len(jobs) {
		printLimit = len(jobs)
	}

	log.Printf("%s %s collected=%d printing=%d", applogger.ColorScope("preview"), applogger.ColorSuccess("READY"), len(jobs), printLimit)
	fmt.Printf("Collected %d jobs.\n\n", len(jobs))

	for i := 0; i < printLimit; i++ {
		output := map[string]any{
			"index":            i + 1,
			"title":            jobs[i].Title,
			"slug":             jobs[i].Slug,
			"company":          jobs[i].Company,
			"location":         jobs[i].Location,
			"employment_type":  jobs[i].EmploymentType,
			"workplace_type":   jobs[i].WorkplaceType,
			"category":         jobs[i].Category,
			"salary_min":       jobs[i].SalaryMin,
			"salary_max":       jobs[i].SalaryMax,
			"currency":         jobs[i].Currency,
			"source_job_url":   jobs[i].SourceJobURL,
			"source_apply_url": jobs[i].SourceApplyURL,
			"posted_at":        formatTime(jobs[i].PostedAt),
			"expired_at":       formatTime(jobs[i].ExpiredAt),
			"description":      jobs[i].Description,
			"requirements":     jobs[i].Requirements,
			"benefits":         jobs[i].Benefits,
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("marshal preview output: %v", err)
		}

		fmt.Println(string(data))
		fmt.Println()
	}

	log.Printf("%s %s", applogger.ColorScope("preview"), applogger.ColorSuccess("DONE"))
}

func buildPreviewConfig(sourceName string, pageLimit int) (models.Source, collector.Collector, collector.SourceScraper) {
	switch strings.ToLower(strings.TrimSpace(sourceName)) {
	case "", "dealls":
		return models.Source{
				Name:    "dealls",
				BaseURL: "https://dealls.com/",
				Mode:    collector.ModeHTTP,
				Active:  true,
			},
			httpcollector.NewWithTimeout(2 * time.Minute),
			sources.NewDeallsScraperWithMaxPages(pageLimit)
	case "glints":
		return models.Source{
				Name:    "glints",
				BaseURL: "https://glints.com/",
				Mode:    collector.ModeBrowser,
				Active:  true,
			},
			browsercollector.New(),
			sources.NewGlintsScraperWithMaxPages(pageLimit)
	default:
		log.Fatalf("unsupported collector preview source %q", sourceName)
	}

	return models.Source{}, nil, nil
}

func getEnvIntWithLegacy(primaryKey, legacyKey string, fallback int) int {
	if value, ok := parseEnvInt(primaryKey); ok {
		return value
	}

	if value, ok := parseEnvInt(legacyKey); ok {
		return value
	}

	return fallback
}

func parseEnvInt(key string) (int, bool) {
	value := os.Getenv(key)
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	if parsed <= 0 {
		return 0, false
	}

	return parsed, true
}

func getEnvString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func formatTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UTC().Format(time.RFC3339)
}
