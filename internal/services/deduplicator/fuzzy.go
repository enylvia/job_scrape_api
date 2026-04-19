package deduplicator

import (
	"fmt"
	"regexp"
	"strings"

	"job_aggregator/internal/models"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func detectFuzzyDuplicate(candidate, existing models.Job) (bool, float64, string) {
	titleScore := similarityScore(candidate.Title, existing.Title)
	companyScore := similarityScore(candidate.Company, existing.Company)
	locationScore := similarityScore(candidate.Location, existing.Location)

	overall := (titleScore * 0.6) + (companyScore * 0.25) + (locationScore * 0.15)

	if titleScore >= 0.92 && companyScore >= 0.95 && locationScore >= 0.80 {
		return true, overall, fmt.Sprintf(
			"fuzzy duplicate by title/company/location score=%.2f (title=%.2f company=%.2f location=%.2f)",
			overall,
			titleScore,
			companyScore,
			locationScore,
		)
	}

	if overall >= 0.93 && companyScore >= 0.90 {
		return true, overall, fmt.Sprintf(
			"fuzzy duplicate by similarity score=%.2f (title=%.2f company=%.2f location=%.2f)",
			overall,
			titleScore,
			companyScore,
			locationScore,
		)
	}

	return false, overall, ""
}

func similarityScore(left, right string) float64 {
	leftTokens := tokenize(left)
	rightTokens := tokenize(right)

	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}

	leftSet := make(map[string]struct{}, len(leftTokens))
	for _, token := range leftTokens {
		leftSet[token] = struct{}{}
	}

	rightSet := make(map[string]struct{}, len(rightTokens))
	for _, token := range rightTokens {
		rightSet[token] = struct{}{}
	}

	intersection := 0
	for token := range leftSet {
		if _, ok := rightSet[token]; ok {
			intersection++
		}
	}

	union := len(leftSet)
	for token := range rightSet {
		if _, ok := leftSet[token]; !ok {
			union++
		}
	}

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func tokenize(value string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil
	}

	return tokenPattern.FindAllString(value, -1)
}

func duplicatePrimary(candidate, existing models.Job) models.Job {
	if candidate.ID < existing.ID {
		return candidate
	}

	return existing
}
