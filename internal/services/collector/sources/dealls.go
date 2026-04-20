package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

const (
	deallsName            = "dealls"
	deallsListEndpoint    = "https://api.sejutacita.id/v1/explore-job/job"
	deallsDetailBaseURL   = "https://dealls.com/loker/"
	deallsDefaultPageSize = 18
	deallsMaxPages        = 3
	deallsDetailTimeout   = 20 * time.Second
)

var nextDataScriptPattern = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json"[^>]*>(.*?)</script>`)

type DeallsScraper struct {
	maxPages int
}

type deallsListResponse struct {
	Data struct {
		Docs []deallsListJob `json:"docs"`
		Page int             `json:"page"`
	} `json:"data"`
}

type deallsListJob struct {
	ID                       string   `json:"id"`
	Slug                     string   `json:"slug"`
	Role                     string   `json:"role"`
	CategorySlug             string   `json:"categorySlug"`
	JobRoleCategorySlug      string   `json:"jobRoleCategorySlug"`
	EmploymentTypes          []string `json:"employmentTypes"`
	WorkplaceType            string   `json:"workplaceType"`
	PublishedAt              string   `json:"publishedAt"`
	ExternalPlatformApplyURL string   `json:"externalPlatformApplyUrl"`
	SalaryType               string   `json:"salaryType"`
	SalaryRange              struct {
		Start *int64 `json:"start"`
		End   *int64 `json:"end"`
	} `json:"salaryRange"`
	Company struct {
		Name              string `json:"name"`
		Slug              string `json:"slug"`
		ProfileImageURL   string `json:"profileImageUrl"`
		ProfilePictureURL string `json:"profilePictureUrl"`
		LogoURL           string `json:"logoUrl"`
		ImageURL          string `json:"imageUrl"`
		Insight           struct {
			Benefits []string `json:"benefits"`
		} `json:"insight"`
	} `json:"company"`
	City struct {
		Name string `json:"name"`
	} `json:"city"`
}

type deallsNextData struct {
	Props struct {
		PageProps struct {
			DehydratedState struct {
				Queries []struct {
					State struct {
						Data deallsDetailJob `json:"data"`
					} `json:"state"`
				} `json:"queries"`
			} `json:"dehydratedState"`
		} `json:"pageProps"`
	} `json:"props"`
}

type deallsDetailJob struct {
	Role                     string `json:"role"`
	Slug                     string `json:"slug"`
	Responsibilities         string `json:"responsibilities"`
	Requirements             string `json:"requirements"`
	Benefits                 string `json:"benefits"`
	EmploymentType           string `json:"employmentType"`
	WorkplaceType            string `json:"workplaceType"`
	PublishedAt              string `json:"publishedAt"`
	ExpiredAt                string `json:"expiredAt"`
	ExternalPlatformApplyURL string `json:"externalPlatformApplyUrl"`
	SalaryRange              struct {
		Start *int64 `json:"start"`
		End   *int64 `json:"end"`
	} `json:"salaryRange"`
	JobRole struct {
		Name string `json:"name"`
	} `json:"jobRole"`
	JobRoleCategory struct {
		Name string `json:"name"`
	} `json:"jobRoleCategory"`
	JobRoleSubCategory struct {
		Name string `json:"name"`
	} `json:"jobRoleSubCategory"`
	Company struct {
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
	} `json:"company"`
	City struct {
		Name string `json:"name"`
	} `json:"city"`
}

func NewDeallsScraper() *DeallsScraper {
	return NewDeallsScraperWithMaxPages(deallsMaxPages)
}

func NewDeallsScraperWithMaxPages(maxPages int) *DeallsScraper {
	if maxPages <= 0 {
		maxPages = deallsMaxPages
	}

	return &DeallsScraper{
		maxPages: maxPages,
	}
}

func (s *DeallsScraper) Name() string {
	return deallsName
}

func (s *DeallsScraper) Collect(ctx context.Context, source models.Source, fetcher collector.Fetcher) ([]collector.CollectedJob, error) {
	log.Printf("%s %s source=%s max_pages=%d", applogger.ColorScope("dealls"), applogger.ColorStart("START"), source.Name, s.maxPages)

	jobs := make([]collector.CollectedJob, 0)
	failedDetails := 0

	for page := 1; page <= s.maxPages; page++ {
		listURL := s.listURL(page)
		log.Printf("%s %s list page=%d url=%s", applogger.ColorScope("dealls"), applogger.ColorFetch("LIST"), page, listURL)
		listResult, err := fetcher.Fetch(ctx, listURL)
		if err != nil {
			return nil, fmt.Errorf("fetch dealls list page %d: %w", page, err)
		}

		var payload deallsListResponse
		if err := json.Unmarshal([]byte(listResult.Body), &payload); err != nil {
			return nil, fmt.Errorf("decode dealls list page %d: %w", page, err)
		}

		if len(payload.Data.Docs) == 0 {
			log.Printf("%s %s page=%d returned no jobs", applogger.ColorScope("dealls"), applogger.ColorWarn("EMPTY"), page)
			break
		}

		log.Printf("%s %s page=%d jobs=%d", applogger.ColorScope("dealls"), applogger.ColorSuccess("LIST_OK"), page, len(payload.Data.Docs))

		for _, item := range payload.Data.Docs {
			detailURL := buildDeallsDetailURL(item.Slug, item.Company.Slug)
			log.Printf("%s %s title=%q url=%s", applogger.ColorScope("dealls"), applogger.ColorFetch("DETAIL"), item.Role, detailURL)

			detailCtx, cancel := context.WithTimeout(ctx, deallsDetailTimeout)
			collected, err := s.collectDetail(detailCtx, fetcher, item, detailURL)
			cancel()
			if err != nil {
				failedDetails++
				log.Printf("%s %s title=%q url=%s err=%v", applogger.ColorScope("dealls"), applogger.ColorWarn("SKIP"), item.Role, detailURL, err)
				continue
			}

			log.Printf("%s %s title=%q company=%q apply_url=%q", applogger.ColorScope("dealls"), applogger.ColorSuccess("PARSED"), collected.Title, collected.Company, collected.SourceApplyURL)
			jobs = append(jobs, collected)
		}
	}

	if len(jobs) == 0 && failedDetails > 0 {
		return nil, fmt.Errorf("all detail page fetches failed for source %s", source.Name)
	}

	log.Printf("%s %s total_jobs=%d failed_details=%d", applogger.ColorScope("dealls"), applogger.ColorSuccess("DONE"), len(jobs), failedDetails)

	return jobs, nil
}

func (s *DeallsScraper) listURL(page int) string {
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("sortParam", "mostRelevant")
	query.Set("sortBy", "asc")
	query.Set("boostTheBoostedJob", "true")
	query.Set("published", "true")
	query.Set("limit", fmt.Sprintf("%d", deallsDefaultPageSize))
	query.Set("status", "active")
	query.Set("externalPlatformApplyUrlSet", "null")

	return deallsListEndpoint + "?" + query.Encode()
}

func (s *DeallsScraper) collectDetail(ctx context.Context, fetcher collector.Fetcher, item deallsListJob, detailURL string) (collector.CollectedJob, error) {
	detailResult, err := fetcher.Fetch(ctx, detailURL)
	if err != nil {
		return collector.CollectedJob{}, fmt.Errorf("fetch detail page: %w", err)
	}

	detailData, err := parseDeallsNextData(detailResult.Body)
	if err != nil {
		return collector.CollectedJob{}, fmt.Errorf("parse embedded detail JSON: %w", err)
	}

	descriptionHTML := firstNonEmpty(detailData.Responsibilities, detailData.Requirements)
	requirementsHTML := detailData.Requirements
	benefitsText := collectBenefits(detailData, item)

	postedAt := parseTimePointer(firstNonEmpty(detailData.PublishedAt, item.PublishedAt))
	expiredAt := parseTimePointer(detailData.ExpiredAt)

	applyURL := firstNonEmpty(
		detailData.ExternalPlatformApplyURL,
		item.ExternalPlatformApplyURL,
		extractFirstExternalLink(detailData.Responsibilities),
		extractFirstExternalLink(detailData.Requirements),
		detailURL,
	)

	rawJSON, err := marshalRawJSON(item, detailData)
	if err != nil {
		return collector.CollectedJob{}, fmt.Errorf("marshal dealls raw json: %w", err)
	}

	return collector.CollectedJob{
		SourceJobURL:   detailURL,
		SourceApplyURL: applyURL,
		Title:          firstNonEmpty(detailData.Role, item.Role),
		Slug:           firstNonEmpty(detailData.Slug, item.Slug),
		Company:        firstNonEmpty(detailData.Company.Name, item.Company.Name),
		CompanyProfileImageURL: firstNonEmpty(
			detailData.Company.ProfileImageURL,
			detailData.Company.ProfilePictureURL,
			detailData.Company.LogoURL,
			detailData.Company.ImageURL,
			item.Company.ProfileImageURL,
			item.Company.ProfilePictureURL,
			item.Company.LogoURL,
			item.Company.ImageURL,
		),
		Location: firstNonEmpty(
			detailData.City.Name,
			detailData.Company.Location.City.Name,
			item.City.Name,
		),
		EmploymentType: normalizeEmploymentType(firstNonEmpty(detailData.EmploymentType, firstEmploymentType(item.EmploymentTypes))),
		WorkplaceType:  normalizeWorkplaceType(firstNonEmpty(detailData.WorkplaceType, item.WorkplaceType)),
		Category:       deriveCategory(detailData, item),
		SalaryMin:      firstNonNilInt64(detailData.SalaryRange.Start, item.SalaryRange.Start),
		SalaryMax:      firstNonNilInt64(detailData.SalaryRange.End, item.SalaryRange.End),
		Currency:       deriveCurrency(item),
		Description:    htmlToText(descriptionHTML),
		Requirements:   htmlToText(requirementsHTML),
		Benefits:       benefitsText,
		PostedAt:       postedAt,
		ExpiredAt:      expiredAt,
		RawHTML:        detailResult.Body,
		RawJSON:        rawJSON,
		CollectedAt:    detailResult.FetchedAt,
	}, nil
}

func parseDeallsNextData(body string) (deallsDetailJob, error) {
	matches := nextDataScriptPattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return deallsDetailJob{}, fmt.Errorf("__NEXT_DATA__ script not found")
	}

	var payload deallsNextData
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return deallsDetailJob{}, fmt.Errorf("decode __NEXT_DATA__: %w", err)
	}

	if len(payload.Props.PageProps.DehydratedState.Queries) == 0 {
		return deallsDetailJob{}, fmt.Errorf("detail queries payload is empty")
	}

	return payload.Props.PageProps.DehydratedState.Queries[0].State.Data, nil
}

func buildDeallsDetailURL(slug, companySlug string) string {
	if strings.TrimSpace(companySlug) == "" {
		return deallsDetailBaseURL + strings.TrimSpace(slug)
	}

	return deallsDetailBaseURL + strings.TrimSpace(slug) + "~" + strings.TrimSpace(companySlug)
}

func firstEmploymentType(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func normalizeEmploymentType(value string) string {
	switch strings.TrimSpace(value) {
	case "fullTime":
		return "full_time"
	case "FULL_TIME":
		return "full_time"
	case "partTime":
		return "part_time"
	case "PART_TIME":
		return "part_time"
	case "contract":
		return "contract"
	case "CONTRACT":
		return "contract"
	case "internship":
		return "internship"
	case "INTERNSHIP":
		return "internship"
	case "freelance":
		return "freelance"
	case "FREELANCE":
		return "freelance"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func normalizeWorkplaceType(value string) string {
	switch strings.TrimSpace(value) {
	case "onSite":
		return "wfo"
	case "ONSITE":
		return "wfo"
	case "hybrid":
		return "hybrid"
	case "HYBRID":
		return "hybrid"
	case "remote":
		return "remote"
	case "REMOTE":
		return "remote"
	default:
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "on_site" || normalized == "onsite" || normalized == "wfo" {
			return "wfo"
		}
		return normalized
	}
}

func deriveCategory(detail deallsDetailJob, item deallsListJob) string {
	return firstNonEmpty(
		detail.JobRoleSubCategory.Name,
		detail.JobRoleCategory.Name,
		detail.JobRole.Name,
		item.JobRoleCategorySlug,
		item.CategorySlug,
	)
}

func deriveCurrency(item deallsListJob) string {
	if item.SalaryRange.Start != nil || item.SalaryRange.End != nil {
		return "IDR"
	}

	return ""
}

func firstNonNilInt64(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}

	return nil
}

func collectBenefits(detail deallsDetailJob, item deallsListJob) string {
	benefits := make([]string, 0)
	benefits = append(benefits, splitBenefitLines(detail.Benefits)...)
	benefits = append(benefits, detail.Company.Insight.Benefits...)
	benefits = append(benefits, item.Company.Insight.Benefits...)

	unique := make([]string, 0, len(benefits))
	seen := map[string]struct{}{}
	for _, benefit := range benefits {
		benefit = strings.TrimSpace(benefit)
		if benefit == "" {
			continue
		}
		if _, ok := seen[benefit]; ok {
			continue
		}
		seen[benefit] = struct{}{}
		unique = append(unique, benefit)
	}

	return strings.Join(unique, "\n")
}

func splitBenefitLines(value string) []string {
	value = htmlToText(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}

	return out
}

func htmlToText(fragment string) string {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + fragment + "</div>"))
	if err != nil {
		return html.UnescapeString(fragment)
	}

	lines := make([]string, 0)
	doc.Find("p, li").Each(func(_ int, selection *goquery.Selection) {
		text := strings.TrimSpace(selection.Text())
		if text != "" {
			lines = append(lines, html.UnescapeString(text))
		}
	})

	if len(lines) > 0 {
		return strings.Join(lines, "\n")
	}

	return strings.TrimSpace(html.UnescapeString(doc.Text()))
}

func extractFirstExternalLink(fragment string) string {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + fragment + "</div>"))
	if err != nil {
		return ""
	}

	link := ""
	doc.Find("a").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href, ok := selection.Attr("href")
		if !ok {
			return true
		}
		href = strings.TrimSpace(href)
		if href == "" {
			return true
		}
		if strings.Contains(href, "dealls.com") {
			return true
		}
		link = href
		return false
	})

	return link
}

func marshalRawJSON(item deallsListJob, detail deallsDetailJob) (string, error) {
	payload := map[string]any{
		"list_job":   item,
		"detail_job": detail,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func parseTimePointer(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return &parsed
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
