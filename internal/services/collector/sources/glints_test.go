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

func TestParseGlintsListNextData(t *testing.T) {
	html := buildGlintsListHTML(glintsSearchResults{
		HasMore: true,
		JobsInPage: []glintsListJob{
			{
				ID:     "job-1",
				Title:  "Backend Engineer",
				Type:   "FULL_TIME",
				Status: "OPEN",
			},
		},
	})

	results, err := parseGlintsListNextData(html)
	if err != nil {
		t.Fatalf("parseGlintsListNextData returned error: %v", err)
	}

	if len(results.JobsInPage) != 1 {
		t.Fatalf("expected 1 job, got %d", len(results.JobsInPage))
	}

	if results.JobsInPage[0].Title != "Backend Engineer" {
		t.Fatalf("unexpected title: %s", results.JobsInPage[0].Title)
	}
}

func TestParseGlintsDetailNextData(t *testing.T) {
	html := buildGlintsDetailHTML(sampleGlintsDetailData())

	detail, err := parseGlintsDetailNextData(html)
	if err != nil {
		t.Fatalf("parseGlintsDetailNextData returned error: %v", err)
	}

	if detail.Job.Title != "Programmer (Magang)" {
		t.Fatalf("unexpected title: %s", detail.Job.Title)
	}

	if detail.Company.Name != "ID-Networkers" {
		t.Fatalf("unexpected company: %s", detail.Company.Name)
	}

	if detail.Location == nil || detail.Location.FormattedName != "Palmerah" {
		t.Fatalf("unexpected location: %+v", detail.Location)
	}
}

func TestGlintsCollectDetailMapsProductionFields(t *testing.T) {
	scraper := NewGlintsScraper()
	item := sampleGlintsListJob()
	detailURL := buildGlintsDetailURL(item.Title, item.ID)

	fetcher := &stubFetcher{
		results: map[string]collector.FetchResult{
			detailURL: {
				URL:       detailURL,
				Body:      buildGlintsDetailHTML(sampleGlintsDetailData()),
				FetchedAt: time.Now().UTC(),
			},
		},
	}

	job, err := scraper.collectDetail(context.Background(), fetcher, item, detailURL)
	if err != nil {
		t.Fatalf("collectDetail returned error: %v", err)
	}

	if job.Title != "Programmer (Magang)" {
		t.Fatalf("unexpected title: %s", job.Title)
	}

	if job.EmploymentType != "internship" {
		t.Fatalf("unexpected employment type: %s", job.EmploymentType)
	}

	if job.WorkplaceType != "wfo" {
		t.Fatalf("unexpected workplace type: %s", job.WorkplaceType)
	}

	if job.Category != "Backend Developer" {
		t.Fatalf("unexpected category: %s", job.Category)
	}

	if job.SalaryMin == nil || *job.SalaryMin != 1000000 {
		t.Fatalf("unexpected salary min: %+v", job.SalaryMin)
	}

	if job.SalaryMax == nil || *job.SalaryMax != 1100000 {
		t.Fatalf("unexpected salary max: %+v", job.SalaryMax)
	}

	if job.Currency != "IDR" {
		t.Fatalf("unexpected currency: %s", job.Currency)
	}

	if !strings.Contains(job.Requirements, "Education: Bachelor's degree") {
		t.Fatalf("requirements missing education line: %q", job.Requirements)
	}

	if !strings.Contains(job.Requirements, "Skills: Node.js, PostgreSQL") {
		t.Fatalf("requirements missing skills line: %q", job.Requirements)
	}

	if !strings.Contains(job.Description, "ID-Networkers was established in 2008") {
		t.Fatalf("unexpected description: %q", job.Description)
	}

	if job.SourceApplyURL != detailURL {
		t.Fatalf("expected source apply url fallback to detail page, got %q", job.SourceApplyURL)
	}
}

func TestGlintsCollectSkipsClosedJobs(t *testing.T) {
	scraper := NewGlintsScraperWithMaxPages(1)
	listURL := scraper.listURL(1)

	listPayload := glintsSearchResults{
		HasMore: false,
		JobsInPage: []glintsListJob{
			{
				ID:     "closed-job",
				Title:  "Closed Job",
				Type:   "FULL_TIME",
				Status: "CLOSED",
			},
			sampleGlintsListJob(),
		},
	}

	detailURL := buildGlintsDetailURL("Programmer (Magang)", "0658bdcb-b66f-40d6-b125-9fcc2892821d")
	fetcher := &stubFetcher{
		results: map[string]collector.FetchResult{
			listURL: {
				URL:       listURL,
				Body:      buildGlintsListHTML(listPayload),
				FetchedAt: time.Now().UTC(),
			},
			detailURL: {
				URL:       detailURL,
				Body:      buildGlintsDetailHTML(sampleGlintsDetailData()),
				FetchedAt: time.Now().UTC(),
			},
		},
	}

	jobs, err := scraper.Collect(context.Background(), models.Source{Name: "glints"}, fetcher)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 collected job, got %d", len(jobs))
	}

	if jobs[0].Title != "Programmer (Magang)" {
		t.Fatalf("unexpected collected job title: %s", jobs[0].Title)
	}
}

func buildGlintsListHTML(results glintsSearchResults) string {
	payload := glintsListNextData{}
	payload.Props.PageProps.InitialJobs = results

	bytes, _ := json.Marshal(payload)
	return `<html><body><script id="__NEXT_DATA__" type="application/json">` + string(bytes) + `</script></body></html>`
}

func buildGlintsDetailHTML(data glintsDetailData) string {
	cache := map[string]any{
		"ROOT_QUERY": map[string]any{
			`getJobById({"id":"0658bdcb-b66f-40d6-b125-9fcc2892821d"})`: map[string]any{
				"__ref": "Job:0658bdcb-b66f-40d6-b125-9fcc2892821d",
			},
		},
		"Job:0658bdcb-b66f-40d6-b125-9fcc2892821d":                     data.Job,
		"Company:ac452398-d5f8-48f4-bd19-6702bc4304f1":                 data.Company,
		"HierarchicalLocation:d6bf6eb8-0fc9-4275-ad05-5b690b99b92c":    data.Location,
		"HierarchicalJobCategory:dcf154bc-d6ae-40df-afd7-bf88152235ee": data.Category,
		"JobSalary:81d622e6-e0fb-428e-9817-e93dfa8be0b5":               data.Salaries[0],
		"SkillDetail:nodejs":     data.Skills[0],
		"SkillDetail:postgresql": data.Skills[1],
	}

	payload := glintsDetailNextData{}
	payload.Props.PageProps.ApolloCache = make(map[string]json.RawMessage, len(cache))
	for key, value := range cache {
		bytes, _ := json.Marshal(value)
		payload.Props.PageProps.ApolloCache[key] = bytes
	}

	bytes, _ := json.Marshal(payload)
	return `<html><body><script id="__NEXT_DATA__" type="application/json">` + string(bytes) + `</script></body></html>`
}

func sampleGlintsListJob() glintsListJob {
	return glintsListJob{
		ID:                    "0658bdcb-b66f-40d6-b125-9fcc2892821d",
		Title:                 "Programmer (Magang)",
		Type:                  "INTERNSHIP",
		Status:                "OPEN",
		CreatedAt:             "2026-02-02T08:39:43.336Z",
		WorkArrangementOption: "ONSITE",
		MinYearsOfExperience:  intPtr(1),
		MaxYearsOfExperience:  intPtr(3),
		EducationLevel:        "BACHELOR_DEGREE",
		ShouldShowSalary:      true,
		Company: glintsListCompany{
			ID:      "ac452398-d5f8-48f4-bd19-6702bc4304f1",
			Name:    "ID-Networkers",
			Website: "http://www.idn.id",
			Address: "Jakarta Barat",
		},
		Location: &glintsListLocation{
			ID:            "d6bf6eb8-0fc9-4275-ad05-5b690b99b92c",
			Name:          "Palmerah",
			FormattedName: "Palmerah",
			Parents: []glintsListLocationRef{
				{FormattedName: "Jakarta Barat"},
				{FormattedName: "DKI Jakarta"},
				{FormattedName: "Indonesia"},
			},
		},
		HierarchicalCategory: &glintsListJobCategory{
			ID:    "dcf154bc-d6ae-40df-afd7-bf88152235ee",
			Name:  "Backend Developer",
			Level: 3,
		},
		Salaries: []glintsListSalary{
			{
				MinAmount: int64Ptr(1000000),
				MaxAmount: int64Ptr(1100000),
				Currency:  "IDR",
			},
		},
		Skills: []glintsListSkillRelation{
			{MustHave: true, Skill: glintsSkillBasic{ID: "nodejs", Name: "Node.js"}},
			{MustHave: true, Skill: glintsSkillBasic{ID: "postgresql", Name: "PostgreSQL"}},
		},
	}
}

func sampleGlintsDetailData() glintsDetailData {
	return glintsDetailData{
		Job: glintsDetailJob{
			ID:                    "0658bdcb-b66f-40d6-b125-9fcc2892821d",
			Title:                 "Programmer (Magang)",
			Type:                  "INTERNSHIP",
			Status:                "OPEN",
			CreatedAt:             "2026-02-02T08:39:43.336Z",
			ExpiryDate:            "2026-03-05T00:00:00Z",
			WorkArrangementOption: "ONSITE",
			EducationLevel:        "BACHELOR_DEGREE",
			ShouldShowSalary:      true,
			MinYearsOfExperience:  intPtr(1),
			MaxYearsOfExperience:  intPtr(3),
			DescriptionJSONString: `{"blocks":[{"text":"ID-Networkers was established in 2008."},{"text":"We are currently hiring a Backend Developer."}],"entityMap":{}}`,
			Company: glintsApolloRef{
				Ref: "Company:ac452398-d5f8-48f4-bd19-6702bc4304f1",
			},
			Location: glintsApolloRef{
				Ref: "HierarchicalLocation:d6bf6eb8-0fc9-4275-ad05-5b690b99b92c",
			},
			HierarchicalJobCategory: glintsApolloRef{
				Ref: "HierarchicalJobCategory:dcf154bc-d6ae-40df-afd7-bf88152235ee",
			},
			Salaries: []glintsApolloRef{
				{Ref: "JobSalary:81d622e6-e0fb-428e-9817-e93dfa8be0b5"},
			},
			Skills: []glintsDetailSkillRelation{
				{MustHave: true, Skill: glintsApolloRef{Ref: "SkillDetail:nodejs"}},
				{MustHave: true, Skill: glintsApolloRef{Ref: "SkillDetail:postgresql"}},
			},
			Benefits: nil,
		},
		Company: glintsListCompany{
			ID:      "ac452398-d5f8-48f4-bd19-6702bc4304f1",
			Name:    "ID-Networkers",
			Website: "http://www.idn.id",
			Address: "Jl. Anggrek Rosliana No.12A",
		},
		Location: &glintsListLocation{
			ID:            "d6bf6eb8-0fc9-4275-ad05-5b690b99b92c",
			Name:          "Palmerah",
			FormattedName: "Palmerah",
			Parents: []glintsListLocationRef{
				{FormattedName: "Jakarta Barat"},
				{FormattedName: "DKI Jakarta"},
				{FormattedName: "Indonesia"},
			},
		},
		Category: &glintsListJobCategory{
			ID:    "dcf154bc-d6ae-40df-afd7-bf88152235ee",
			Name:  "Backend Developer",
			Level: 3,
		},
		Salaries: []glintsListSalary{
			{
				MinAmount: int64Ptr(1000000),
				MaxAmount: int64Ptr(1100000),
				Currency:  "IDR",
			},
		},
		Skills: []glintsSkillBasic{
			{ID: "nodejs", Name: "Node.js"},
			{ID: "postgresql", Name: "PostgreSQL"},
		},
	}
}

func intPtr(value int) *int {
	return &value
}
