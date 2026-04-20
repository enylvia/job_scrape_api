package sources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

type stubFetcher struct {
	results map[string]collector.FetchResult
	errors  map[string]error
}

func (s *stubFetcher) Fetch(_ context.Context, url string) (collector.FetchResult, error) {
	if err, ok := s.errors[url]; ok {
		return collector.FetchResult{}, err
	}

	result, ok := s.results[url]
	if !ok {
		return collector.FetchResult{}, context.DeadlineExceeded
	}

	return result, nil
}

func TestParseDeallsNextData(t *testing.T) {
	html := buildDetailHTML(deallsDetailJob{
		Role:             "Backend Engineer",
		Slug:             "backend-engineer",
		Responsibilities: "<p>Build APIs</p>",
		Requirements:     "<p>Know Go</p>",
	})

	detail, err := parseDeallsNextData(html)
	if err != nil {
		t.Fatalf("parseDeallsNextData returned error: %v", err)
	}

	if detail.Role != "Backend Engineer" {
		t.Fatalf("unexpected role: %s", detail.Role)
	}

	if detail.Slug != "backend-engineer" {
		t.Fatalf("unexpected slug: %s", detail.Slug)
	}
}

func TestHTMLToText(t *testing.T) {
	got := htmlToText("<p>Hello</p><ul><li>World</li></ul>")
	want := "Hello\nWorld"

	if got != want {
		t.Fatalf("unexpected htmlToText result:\nwant=%q\ngot=%q", want, got)
	}
}

func TestDeallsCollectContinuesWhenOneDetailFails(t *testing.T) {
	scraper := NewDeallsScraperWithMaxPages(1)
	listURL := scraper.listURL(1)
	goodDetailURL := buildDeallsDetailURL("backend-engineer", "acme")
	badDetailURL := buildDeallsDetailURL("data-engineer", "acme")

	listPayload := deallsListResponse{}
	listPayload.Data.Docs = []deallsListJob{
		{
			ID:              "1",
			Slug:            "backend-engineer",
			Role:            "Backend Engineer",
			EmploymentTypes: []string{"fullTime"},
			WorkplaceType:   "remote",
			PublishedAt:     "2026-04-16T00:00:00Z",
			Company: struct {
				Name              string `json:"name"`
				Slug              string `json:"slug"`
				ProfileImageURL   string `json:"profileImageUrl"`
				ProfilePictureURL string `json:"profilePictureUrl"`
				LogoURL           string `json:"logoUrl"`
				ImageURL          string `json:"imageUrl"`
				Insight           struct {
					Benefits []string `json:"benefits"`
				} `json:"insight"`
			}{
				Name: "Acme",
				Slug: "acme",
			},
			City: struct {
				Name string `json:"name"`
			}{Name: "Jakarta"},
		},
		{
			ID:              "2",
			Slug:            "data-engineer",
			Role:            "Data Engineer",
			EmploymentTypes: []string{"fullTime"},
			WorkplaceType:   "hybrid",
			PublishedAt:     "2026-04-16T00:00:00Z",
			Company: struct {
				Name              string `json:"name"`
				Slug              string `json:"slug"`
				ProfileImageURL   string `json:"profileImageUrl"`
				ProfilePictureURL string `json:"profilePictureUrl"`
				LogoURL           string `json:"logoUrl"`
				ImageURL          string `json:"imageUrl"`
				Insight           struct {
					Benefits []string `json:"benefits"`
				} `json:"insight"`
			}{
				Name: "Acme",
				Slug: "acme",
			},
			City: struct {
				Name string `json:"name"`
			}{Name: "Jakarta"},
		},
	}

	listBytes, _ := json.Marshal(listPayload)
	fetcher := &stubFetcher{
		results: map[string]collector.FetchResult{
			listURL: {
				URL:       listURL,
				Body:      string(listBytes),
				FetchedAt: time.Now().UTC(),
			},
			goodDetailURL: {
				URL:       goodDetailURL,
				Body:      buildDetailHTML(sampleDetailJob()),
				FetchedAt: time.Now().UTC(),
			},
		},
		errors: map[string]error{
			badDetailURL: context.DeadlineExceeded,
		},
	}

	jobs, err := scraper.Collect(context.Background(), models.Source{Name: "dealls"}, fetcher)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 collected job, got %d", len(jobs))
	}

	if jobs[0].Title != "Backend Engineer" {
		t.Fatalf("unexpected collected title: %s", jobs[0].Title)
	}
}

func TestDeallsCollectDetailMapsProductionFields(t *testing.T) {
	scraper := NewDeallsScraper()
	item := deallsListJob{
		ID:                  "1",
		Slug:                "backend-engineer",
		Role:                "Backend Engineer",
		JobRoleCategorySlug: "engineering",
		EmploymentTypes:     []string{"fullTime"},
		WorkplaceType:       "remote",
		PublishedAt:         "2026-04-16T00:00:00Z",
		SalaryRange: struct {
			Start *int64 `json:"start"`
			End   *int64 `json:"end"`
		}{
			Start: int64Ptr(10000000),
			End:   int64Ptr(15000000),
		},
		Company: struct {
			Name              string `json:"name"`
			Slug              string `json:"slug"`
			ProfileImageURL   string `json:"profileImageUrl"`
			ProfilePictureURL string `json:"profilePictureUrl"`
			LogoURL           string `json:"logoUrl"`
			ImageURL          string `json:"imageUrl"`
			Insight           struct {
				Benefits []string `json:"benefits"`
			} `json:"insight"`
		}{
			Name: "Acme",
			Slug: "acme",
			Insight: struct {
				Benefits []string `json:"benefits"`
			}{
				Benefits: []string{"Remote allowance"},
			},
		},
		City: struct {
			Name string `json:"name"`
		}{Name: "Jakarta"},
	}

	detailURL := buildDeallsDetailURL(item.Slug, item.Company.Slug)
	fetcher := &stubFetcher{
		results: map[string]collector.FetchResult{
			detailURL: {
				URL:       detailURL,
				Body:      buildDetailHTML(sampleDetailJob()),
				FetchedAt: time.Now().UTC(),
			},
		},
	}

	job, err := scraper.collectDetail(context.Background(), fetcher, item, detailURL)
	if err != nil {
		t.Fatalf("collectDetail returned error: %v", err)
	}

	if job.Slug != "backend-engineer" {
		t.Fatalf("unexpected slug: %s", job.Slug)
	}

	if job.Category != "Platform Engineering" {
		t.Fatalf("unexpected category: %s", job.Category)
	}

	if job.Currency != "IDR" {
		t.Fatalf("unexpected currency: %s", job.Currency)
	}

	if job.WorkplaceType != "remote" {
		t.Fatalf("unexpected workplace type: %s", job.WorkplaceType)
	}

	if job.SalaryMin == nil || *job.SalaryMin != 10000000 {
		t.Fatalf("unexpected salary min: %+v", job.SalaryMin)
	}

	if job.SalaryMax == nil || *job.SalaryMax != 15000000 {
		t.Fatalf("unexpected salary max: %+v", job.SalaryMax)
	}

	if !strings.Contains(job.Requirements, "Know Go") {
		t.Fatalf("requirements were not normalized as expected: %q", job.Requirements)
	}
}

func buildDetailHTML(detail deallsDetailJob) string {
	payload := deallsNextData{}
	payload.Props.PageProps.DehydratedState.Queries = []struct {
		State struct {
			Data deallsDetailJob `json:"data"`
		} `json:"state"`
	}{
		{},
	}
	payload.Props.PageProps.DehydratedState.Queries[0].State.Data = detail

	bytes, _ := json.Marshal(payload)
	return `<html><body><script id="__NEXT_DATA__" type="application/json">` + string(bytes) + `</script></body></html>`
}

func sampleDetailJob() deallsDetailJob {
	return deallsDetailJob{
		Role:             "Backend Engineer",
		Slug:             "backend-engineer",
		Responsibilities: "<p>Build APIs</p>",
		Requirements:     "<p>Know Go</p>",
		Benefits:         "<p>Health insurance</p>",
		EmploymentType:   "fullTime",
		WorkplaceType:    "remote",
		PublishedAt:      "2026-04-16T00:00:00Z",
		SalaryRange: struct {
			Start *int64 `json:"start"`
			End   *int64 `json:"end"`
		}{
			Start: int64Ptr(10000000),
			End:   int64Ptr(15000000),
		},
		JobRoleCategory: struct {
			Name string `json:"name"`
		}{Name: "Engineering"},
		JobRoleSubCategory: struct {
			Name string `json:"name"`
		}{Name: "Platform Engineering"},
		Company: struct {
			Name              string `json:"name"`
			Website           string `json:"website"`
			ProfileImageURL   string `json:"profileImageUrl"`
			ProfilePictureURL string `json:"profilePictureUrl"`
			LogoURL           string `json:"logoUrl"`
			ImageURL          string `json:"imageUrl"`
			Insight           struct {
				Benefits []string `json:"benefits"`
			} `json:"insight"`
			Location struct {
				City struct {
					Name string `json:"name"`
				} `json:"city"`
			} `json:"location"`
		}{
			Name:    "Acme",
			Website: "https://acme.test",
			Insight: struct {
				Benefits []string `json:"benefits"`
			}{
				Benefits: []string{"Remote allowance"},
			},
			Location: struct {
				City struct {
					Name string `json:"name"`
				} `json:"city"`
			}{
				City: struct {
					Name string `json:"name"`
				}{Name: "Jakarta"},
			},
		},
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
