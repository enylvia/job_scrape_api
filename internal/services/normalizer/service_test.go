package normalizer

import (
	"testing"

	"job_aggregator/internal/enums"
	"job_aggregator/internal/models"
)

func TestNormalizeEmploymentType(t *testing.T) {
	testCases := map[string]string{
		"fullTime":   "full_time",
		"FULL_TIME":  "full_time",
		"part time":  "part_time",
		"contractor": "contract",
		"intern":     "internship",
		"FREELANCE":  "freelance",
	}

	for input, expected := range testCases {
		if actual := normalizeEmploymentType(input); actual != expected {
			t.Fatalf("normalizeEmploymentType(%q) = %q, expected %q", input, actual, expected)
		}
	}
}

func TestGenerateSlug(t *testing.T) {
	value := generateSlug("Senior Backend Engineer (Go) & API")
	if value != "senior-backend-engineer-go-and-api" {
		t.Fatalf("unexpected slug: %s", value)
	}
}

func TestNormalizeJobValid(t *testing.T) {
	job := normalizeJob(models.Job{
		ID:             1,
		SourceJobURL:   " https://example.com/jobs/1 ",
		SourceApplyURL: " https://example.com/jobs/1/apply ",
		Title:          "  Senior Backend Engineer  ",
		Company:        "  Example   Corp ",
		Location:       "Jakarta \n Selatan, Jakarta Selatan",
		EmploymentType: "FULL_TIME",
		Description:    "Lead   backend  systems.\n\nBuild APIs.",
		Requirements:   "Go \n PostgreSQL",
		Benefits:       "Remote\nBonus",
	})

	if job.Status != string(enums.JobStatusNormalized) {
		t.Fatalf("expected status normalized, got %s", job.Status)
	}

	if job.Slug != "senior-backend-engineer" {
		t.Fatalf("unexpected slug: %s", job.Slug)
	}

	if job.Location != "Jakarta, Selatan, Jakarta Selatan" {
		t.Fatalf("unexpected location: %q", job.Location)
	}

	if job.EmploymentType != "full_time" {
		t.Fatalf("unexpected employment type: %s", job.EmploymentType)
	}
}

func TestNormalizeJobInvalidMovesToReviewPending(t *testing.T) {
	job := normalizeJob(models.Job{
		ID:             2,
		SourceJobURL:   "https://example.com/jobs/2",
		EmploymentType: "FULL_TIME",
	})

	if job.Status != string(enums.JobStatusReviewPending) {
		t.Fatalf("expected status review_pending, got %s", job.Status)
	}
}
