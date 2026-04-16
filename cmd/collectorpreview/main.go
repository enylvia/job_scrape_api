package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
	"job_aggregator/internal/services/collector/httpcollector"
	"job_aggregator/internal/services/collector/sources"
)

func main() {
	pageLimit := getEnvIntWithLegacy("COLLECTOR_PREVIEW_PAGES", "DEALLS_PREVIEW_PAGES", 1)
	printLimit := getEnvIntWithLegacy("COLLECTOR_PREVIEW_PRINT_LIMIT", "DEALLS_PREVIEW_PRINT_LIMIT", 3)

	log.Printf("%s %s collector preview (pages=%d print_limit=%d)", applogger.ColorScope("preview"), applogger.ColorStart("START"), pageLimit, printLimit)

	source := models.Source{
		Name:    "dealls",
		BaseURL: "https://dealls.com/",
		Mode:    collector.ModeHTTP,
		Active:  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	engine := httpcollector.NewWithTimeout(2 * time.Minute)
	scraper := sources.NewDeallsScraperWithMaxPages(pageLimit)

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

func formatTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UTC().Format(time.RFC3339)
}
