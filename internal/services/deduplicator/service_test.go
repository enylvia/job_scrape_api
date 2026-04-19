package deduplicator

import (
	"testing"

	"job_aggregator/internal/models"
)

func TestDetectExactDuplicate(t *testing.T) {
	left := models.Job{
		ID:             1,
		SourceJobURL:   "https://example.com/jobs/1",
		SourceApplyURL: "https://example.com/jobs/1/apply",
		ContentHash:    "abc123",
	}
	right := models.Job{
		ID:             2,
		SourceJobURL:   "https://example.com/jobs/1",
		SourceApplyURL: "https://example.com/jobs/2/apply",
		ContentHash:    "zzz999",
	}

	matched, reason := detectExactDuplicate(left, right)
	if !matched {
		t.Fatalf("expected exact duplicate to match")
	}
	if reason == "" {
		t.Fatalf("expected exact duplicate reason")
	}
}

func TestDetectFuzzyDuplicate(t *testing.T) {
	left := models.Job{
		ID:       1,
		Title:    "Senior Backend Engineer",
		Company:  "Example Tech",
		Location: "Jakarta Selatan, DKI Jakarta",
	}
	right := models.Job{
		ID:       2,
		Title:    "Backend Engineer Senior",
		Company:  "Example Tech",
		Location: "Jakarta Selatan",
	}

	matched, score, reason := detectFuzzyDuplicate(left, right)
	if !matched {
		t.Fatalf("expected fuzzy duplicate to match, score=%.2f", score)
	}
	if reason == "" {
		t.Fatalf("expected fuzzy duplicate reason")
	}
}

func TestFindDuplicatePrefersExact(t *testing.T) {
	candidate := models.Job{
		ID:           2,
		SourceJobURL: "https://example.com/jobs/1",
		Title:        "Backend Engineer",
		Company:      "Example Tech",
		Location:     "Jakarta",
	}
	canonicals := []models.Job{
		{
			ID:           1,
			SourceJobURL: "https://example.com/jobs/1",
			Title:        "Backend Engineer",
			Company:      "Example Tech",
			Location:     "Jakarta",
		},
	}

	matched, duplicateOfID, reason := findDuplicate(candidate, canonicals)
	if !matched {
		t.Fatalf("expected duplicate to match")
	}
	if duplicateOfID != 1 {
		t.Fatalf("expected duplicate_of_job_id=1, got %d", duplicateOfID)
	}
	if reason == "" {
		t.Fatalf("expected duplicate reason")
	}
}
