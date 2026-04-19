package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/models"
	"job_aggregator/internal/services/collector"
)

const (
	glintsName          = "glints"
	glintsBaseURL       = "https://glints.com"
	glintsListPath      = "/id/en/opportunities/jobs/explore"
	glintsDetailBaseURL = "https://glints.com/id/en/opportunities/jobs/"
	glintsMaxPages      = 3
	glintsDetailTimeout = 20 * time.Second
)

type GlintsScraper struct {
	maxPages int
}

type glintsListNextData struct {
	Props struct {
		PageProps struct {
			InitialJobs glintsSearchResults `json:"initialJobs"`
		} `json:"pageProps"`
	} `json:"props"`
}

type glintsSearchResults struct {
	HasMore    bool            `json:"hasMore"`
	JobsInPage []glintsListJob `json:"jobsInPage"`
}

type glintsListJob struct {
	ID                    string                    `json:"id"`
	Title                 string                    `json:"title"`
	Type                  string                    `json:"type"`
	Status                string                    `json:"status"`
	CreatedAt             string                    `json:"createdAt"`
	UpdatedAt             string                    `json:"updatedAt"`
	WorkArrangementOption string                    `json:"workArrangementOption"`
	MinYearsOfExperience  *int                      `json:"minYearsOfExperience"`
	MaxYearsOfExperience  *int                      `json:"maxYearsOfExperience"`
	EducationLevel        string                    `json:"educationLevel"`
	ShouldShowSalary      bool                      `json:"shouldShowSalary"`
	ExternalApplyURL      string                    `json:"externalApplyURL"`
	Company               glintsListCompany         `json:"company"`
	Location              *glintsListLocation       `json:"location"`
	HierarchicalCategory  *glintsListJobCategory    `json:"hierarchicalJobCategory"`
	Salaries              []glintsListSalary        `json:"salaries"`
	Skills                []glintsListSkillRelation `json:"skills"`
}

type glintsListCompany struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Brand    string `json:"brandName"`
	Website  string `json:"website"`
	Address  string `json:"address"`
	IsVIP    bool   `json:"isVIP"`
	Industry any    `json:"industry"`
}

type glintsListLocation struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	FormattedName       string                  `json:"formattedName"`
	AdministrativeLevel string                  `json:"administrativeLevelName"`
	Slug                string                  `json:"slug"`
	Parents             []glintsListLocationRef `json:"parents"`
}

type glintsListLocationRef struct {
	FormattedName       string `json:"formattedName"`
	AdministrativeLevel string `json:"administrativeLevelName"`
	Slug                string `json:"slug"`
}

type glintsListJobCategory struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	Slug    string                  `json:"slug"`
	Level   int                     `json:"level"`
	Parents []glintsListCategoryRef `json:"parents"`
}

type glintsListCategoryRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	Slug  string `json:"slug"`
}

type glintsListSalary struct {
	SalaryMode string `json:"salaryMode"`
	MaxAmount  *int64 `json:"maxAmount"`
	MinAmount  *int64 `json:"minAmount"`
	Currency   string `json:"CurrencyCode"`
}

type glintsListSkillRelation struct {
	MustHave bool             `json:"mustHave"`
	Skill    glintsSkillBasic `json:"skill"`
}

type glintsSkillBasic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type glintsDetailNextData struct {
	Props struct {
		PageProps struct {
			ApolloCache map[string]json.RawMessage `json:"apolloCache"`
		} `json:"pageProps"`
	} `json:"props"`
}

type glintsApolloRef struct {
	Ref string `json:"__ref"`
}

type glintsDetailJob struct {
	ID                      string                      `json:"id"`
	Title                   string                      `json:"title"`
	Type                    string                      `json:"type"`
	Status                  string                      `json:"status"`
	CreatedAt               string                      `json:"createdAt"`
	UpdatedAt               string                      `json:"updatedAt"`
	ExpiryDate              string                      `json:"expiryDate"`
	WorkArrangementOption   string                      `json:"workArrangementOption"`
	EducationLevel          string                      `json:"educationLevel"`
	ResumeRequiredStatus    string                      `json:"resumeRequiredStatus"`
	MinYearsOfExperience    *int                        `json:"minYearsOfExperience"`
	MaxYearsOfExperience    *int                        `json:"maxYearsOfExperience"`
	DescriptionJSONString   string                      `json:"descriptionJsonString"`
	InterviewProcessJSON    string                      `json:"interviewProcessJsonString"`
	ExternalApplyURL        string                      `json:"externalApplyURL"`
	ShouldShowSalary        bool                        `json:"shouldShowSalary"`
	ShouldShowBenefits      bool                        `json:"shouldShowBenefits"`
	Company                 glintsApolloRef             `json:"company"`
	Location                glintsApolloRef             `json:"location"`
	HierarchicalJobCategory glintsApolloRef             `json:"hierarchicalJobCategory"`
	Salaries                []glintsApolloRef           `json:"salaries"`
	Skills                  []glintsDetailSkillRelation `json:"skills"`
	Benefits                any                         `json:"benefits"`
}

type glintsDetailSkillRelation struct {
	MustHave bool            `json:"mustHave"`
	Skill    glintsApolloRef `json:"skill"`
}

type glintsDraftJSDocument struct {
	Blocks []struct {
		Text string `json:"text"`
	} `json:"blocks"`
}

type glintsDetailData struct {
	Job      glintsDetailJob
	Company  glintsListCompany
	Location *glintsListLocation
	Category *glintsListJobCategory
	Salaries []glintsListSalary
	Skills   []glintsSkillBasic
}

type glintsStructuredJobPosting struct {
	Type                 string `json:"@type"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	DatePosted           string `json:"datePosted"`
	EmploymentType       string `json:"employmentType"`
	ValidThrough         string `json:"validThrough"`
	DirectApply          bool   `json:"directApply"`
	OccupationalCategory string `json:"occupationalCategory"`
	Skills               string `json:"skills"`
	EmployerOverview     string `json:"employerOverview"`
	HiringOrganization   struct {
		Name string `json:"name"`
	} `json:"hiringOrganization"`
	JobLocation struct {
		Address struct {
			StreetAddress   string `json:"streetAddress"`
			AddressLocality string `json:"addressLocality"`
			AddressRegion   string `json:"addressRegion"`
			AddressCountry  string `json:"addressCountry"`
		} `json:"address"`
	} `json:"jobLocation"`
	EducationRequirements struct {
		CredentialCategory string `json:"credentialCategory"`
	} `json:"educationRequirements"`
	BaseSalary struct {
		Currency string `json:"currency"`
		Value    struct {
			MinValue *int64 `json:"minValue"`
			MaxValue *int64 `json:"maxValue"`
		} `json:"value"`
	} `json:"baseSalary"`
}

type glintsStructuredSections struct {
	Requirements string
	Description  string
	Benefits     string
}

func NewGlintsScraper() *GlintsScraper {
	return NewGlintsScraperWithMaxPages(glintsMaxPages)
}

func NewGlintsScraperWithMaxPages(maxPages int) *GlintsScraper {
	if maxPages <= 0 {
		maxPages = glintsMaxPages
	}

	return &GlintsScraper{maxPages: maxPages}
}

func (s *GlintsScraper) Name() string {
	return glintsName
}

func (s *GlintsScraper) Collect(ctx context.Context, source models.Source, fetcher collector.Fetcher) ([]collector.CollectedJob, error) {
	log.Printf("%s %s source=%s max_pages=%d", applogger.ColorScope("glints"), applogger.ColorStart("START"), source.Name, s.maxPages)

	jobs := make([]collector.CollectedJob, 0)
	failedDetails := 0

	for page := 1; page <= s.maxPages; page++ {
		listURL := s.listURL(page)
		log.Printf("%s %s page=%d url=%s", applogger.ColorScope("glints"), applogger.ColorFetch("LIST"), page, listURL)

		listResult, err := fetcher.Fetch(ctx, listURL)
		if err != nil {
			return nil, fmt.Errorf("fetch glints list page %d: %w", page, err)
		}

		searchResults, err := parseGlintsListNextData(listResult.Body)
		if err != nil {
			return nil, fmt.Errorf("parse glints list page %d: %w", page, err)
		}

		if len(searchResults.JobsInPage) == 0 {
			log.Printf("%s %s page=%d returned no jobs", applogger.ColorScope("glints"), applogger.ColorWarn("EMPTY"), page)
			break
		}

		log.Printf("%s %s page=%d jobs=%d", applogger.ColorScope("glints"), applogger.ColorSuccess("LIST_OK"), page, len(searchResults.JobsInPage))

		for _, item := range searchResults.JobsInPage {
			if !strings.EqualFold(item.Status, "OPEN") {
				log.Printf("%s %s title=%q status=%s", applogger.ColorScope("glints"), applogger.ColorWarn("SKIP"), item.Title, item.Status)
				continue
			}

			detailURL := buildGlintsDetailURL(item.Title, item.ID)
			log.Printf("%s %s title=%q url=%s", applogger.ColorScope("glints"), applogger.ColorFetch("DETAIL"), item.Title, detailURL)

			detailCtx, cancel := context.WithTimeout(ctx, glintsDetailTimeout)
			collected, err := s.collectDetail(detailCtx, fetcher, item, detailURL)
			cancel()
			if err != nil {
				failedDetails++
				log.Printf("%s %s title=%q url=%s err=%v", applogger.ColorScope("glints"), applogger.ColorWarn("SKIP"), item.Title, detailURL, err)
				continue
			}

			log.Printf("%s %s title=%q company=%q apply_url=%q", applogger.ColorScope("glints"), applogger.ColorSuccess("PARSED"), collected.Title, collected.Company, collected.SourceApplyURL)
			jobs = append(jobs, collected)
		}

		if !searchResults.HasMore {
			break
		}
	}

	if len(jobs) == 0 && failedDetails > 0 {
		return nil, fmt.Errorf("all detail page fetches failed for source %s", source.Name)
	}

	log.Printf("%s %s total_jobs=%d failed_details=%d", applogger.ColorScope("glints"), applogger.ColorSuccess("DONE"), len(jobs), failedDetails)
	return jobs, nil
}

func (s *GlintsScraper) listURL(page int) string {
	query := url.Values{}
	query.Set("country", "ID")
	query.Set("locationName", "All Cities/Provinces")
	if page > 1 {
		query.Set("page", fmt.Sprintf("%d", page))
	}

	return glintsBaseURL + glintsListPath + "?" + query.Encode()
}

func (s *GlintsScraper) collectDetail(ctx context.Context, fetcher collector.Fetcher, item glintsListJob, detailURL string) (collector.CollectedJob, error) {
	detailResult, err := fetcher.Fetch(ctx, detailURL)
	if err != nil {
		return collector.CollectedJob{}, fmt.Errorf("fetch detail page: %w", err)
	}

	detailData, detailErr := parseGlintsDetailNextData(detailResult.Body)
	structured, structuredErr := parseGlintsStructuredJobPosting(detailResult.Body)
	if detailErr != nil && structuredErr != nil {
		return collector.CollectedJob{}, fmt.Errorf("parse detail page: next_data=%v structured_data=%v", detailErr, structuredErr)
	}

	structuredSections := extractGlintsStructuredSections(structured.Description)
	description := firstNonEmpty(
		renderGlintsDraftText(detailData.Job.DescriptionJSONString),
		structuredSections.Description,
		htmlToText(structured.Description),
	)
	if overview := htmlToText(structured.EmployerOverview); overview != "" && !strings.Contains(description, overview) {
		description = strings.TrimSpace(description + "\n" + overview)
	}

	requirements := strings.TrimSpace(strings.Join([]string{
		structuredSections.Requirements,
		buildGlintsRequirements(detailData.Job, detailData.Skills, structured),
	}, "\n"))
	requirements = strings.TrimSpace(strings.ReplaceAll(requirements, "\n\n", "\n"))

	benefits := firstNonEmpty(
		normalizeGlintsBenefits(detailData.Job.Benefits),
		structuredSections.Benefits,
	)
	category := firstNonEmpty(
		deriveGlintsCategory(detailData.Category),
		strings.TrimSpace(structured.OccupationalCategory),
		deriveGlintsCategory(item.HierarchicalCategory),
	)
	location := firstNonEmpty(
		deriveGlintsLocation(detailData.Location),
		deriveGlintsStructuredLocation(structured),
		deriveGlintsLocation(item.Location),
	)
	salaryMin, salaryMax, currency := deriveGlintsSalary(detailData.Job.ShouldShowSalary, detailData.Salaries)
	if salaryMin == nil && salaryMax == nil && currency == "" {
		salaryMin, salaryMax, currency = deriveGlintsStructuredSalary(structured)
	}

	applyURL := firstNonEmpty(
		detailData.Job.ExternalApplyURL,
		item.ExternalApplyURL,
		detailURL,
	)

	rawJSON, err := marshalGlintsRawJSON(item, detailData)
	if err != nil {
		return collector.CollectedJob{}, fmt.Errorf("marshal glints raw json: %w", err)
	}

	return collector.CollectedJob{
		SourceJobURL:   detailURL,
		SourceApplyURL: applyURL,
		Title:          firstNonEmpty(detailData.Job.Title, structured.Title, item.Title),
		Slug:           glintsSlugFromTitle(firstNonEmpty(detailData.Job.Title, item.Title)),
		Company: firstNonEmpty(
			detailData.Company.Brand,
			detailData.Company.Name,
			structured.HiringOrganization.Name,
			item.Company.Brand,
			item.Company.Name,
		),
		Location:       location,
		EmploymentType: normalizeEmploymentType(firstNonEmpty(detailData.Job.Type, structured.EmploymentType, item.Type)),
		WorkplaceType:  normalizeWorkplaceType(firstNonEmpty(detailData.Job.WorkArrangementOption, item.WorkArrangementOption)),
		Category:       category,
		SalaryMin:      salaryMin,
		SalaryMax:      salaryMax,
		Currency:       currency,
		Description:    description,
		Requirements:   requirements,
		Benefits:       benefits,
		PostedAt:       parseTimePointer(firstNonEmpty(detailData.Job.CreatedAt, structured.DatePosted, item.CreatedAt)),
		ExpiredAt:      parseTimePointer(firstNonEmpty(detailData.Job.ExpiryDate, structured.ValidThrough)),
		RawHTML:        detailResult.Body,
		RawJSON:        rawJSON,
		CollectedAt:    detailResult.FetchedAt,
	}, nil
}

func parseGlintsListNextData(body string) (glintsSearchResults, error) {
	matches := nextDataScriptPattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return glintsSearchResults{}, fmt.Errorf("__NEXT_DATA__ script not found")
	}

	var payload glintsListNextData
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return glintsSearchResults{}, fmt.Errorf("decode __NEXT_DATA__: %w", err)
	}

	return payload.Props.PageProps.InitialJobs, nil
}

func parseGlintsDetailNextData(body string) (glintsDetailData, error) {
	matches := nextDataScriptPattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return glintsDetailData{}, fmt.Errorf("__NEXT_DATA__ script not found")
	}

	var payload glintsDetailNextData
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return glintsDetailData{}, fmt.Errorf("decode __NEXT_DATA__: %w", err)
	}

	cache := payload.Props.PageProps.ApolloCache
	if len(cache) == 0 {
		return glintsDetailData{}, fmt.Errorf("apollo cache is empty")
	}

	rootRaw, ok := cache["ROOT_QUERY"]
	if !ok {
		return glintsDetailData{}, fmt.Errorf("ROOT_QUERY not found")
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(rootRaw, &root); err != nil {
		return glintsDetailData{}, fmt.Errorf("decode ROOT_QUERY: %w", err)
	}

	var jobRef glintsApolloRef
	found := false
	for key, raw := range root {
		if strings.HasPrefix(key, "getJobById(") {
			if err := json.Unmarshal(raw, &jobRef); err != nil {
				return glintsDetailData{}, fmt.Errorf("decode job reference: %w", err)
			}
			found = true
			break
		}
	}
	if !found || jobRef.Ref == "" {
		return glintsDetailData{}, fmt.Errorf("job reference not found in ROOT_QUERY")
	}

	var job glintsDetailJob
	if err := unmarshalApolloEntity(cache, jobRef.Ref, &job); err != nil {
		return glintsDetailData{}, fmt.Errorf("decode job entity: %w", err)
	}

	company, err := resolveGlintsCompany(cache, job.Company)
	if err != nil {
		return glintsDetailData{}, err
	}
	location, err := resolveGlintsLocation(cache, job.Location)
	if err != nil {
		return glintsDetailData{}, err
	}
	category, err := resolveGlintsCategory(cache, job.HierarchicalJobCategory)
	if err != nil {
		return glintsDetailData{}, err
	}
	salaries, err := resolveGlintsSalaries(cache, job.Salaries)
	if err != nil {
		return glintsDetailData{}, err
	}
	skills, err := resolveGlintsSkills(cache, job.Skills)
	if err != nil {
		return glintsDetailData{}, err
	}

	return glintsDetailData{
		Job:      job,
		Company:  company,
		Location: location,
		Category: category,
		Salaries: salaries,
		Skills:   skills,
	}, nil
}

func parseGlintsStructuredJobPosting(body string) (glintsStructuredJobPosting, error) {
	matches := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return glintsStructuredJobPosting{}, fmt.Errorf("job posting structured data not found")
	}

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		var posting glintsStructuredJobPosting
		if err := json.Unmarshal([]byte(match[1]), &posting); err != nil {
			continue
		}
		if posting.Type == "JobPosting" {
			return posting, nil
		}
	}

	return glintsStructuredJobPosting{}, fmt.Errorf("job posting structured data not found")
}

func resolveGlintsCompany(cache map[string]json.RawMessage, ref glintsApolloRef) (glintsListCompany, error) {
	if ref.Ref == "" {
		return glintsListCompany{}, nil
	}

	var company glintsListCompany
	if err := unmarshalApolloEntity(cache, ref.Ref, &company); err != nil {
		return glintsListCompany{}, err
	}

	return company, nil
}

func resolveGlintsLocation(cache map[string]json.RawMessage, ref glintsApolloRef) (*glintsListLocation, error) {
	if ref.Ref == "" {
		return nil, nil
	}

	var location glintsListLocation
	if err := unmarshalApolloEntity(cache, ref.Ref, &location); err != nil {
		return nil, err
	}

	return &location, nil
}

func resolveGlintsCategory(cache map[string]json.RawMessage, ref glintsApolloRef) (*glintsListJobCategory, error) {
	if ref.Ref == "" {
		return nil, nil
	}

	var category glintsListJobCategory
	if err := unmarshalApolloEntity(cache, ref.Ref, &category); err != nil {
		return nil, err
	}

	return &category, nil
}

func resolveGlintsSalaries(cache map[string]json.RawMessage, refs []glintsApolloRef) ([]glintsListSalary, error) {
	salaries := make([]glintsListSalary, 0, len(refs))
	for _, ref := range refs {
		if ref.Ref == "" {
			continue
		}

		var salary glintsListSalary
		if err := unmarshalApolloEntity(cache, ref.Ref, &salary); err != nil {
			return nil, err
		}
		salaries = append(salaries, salary)
	}

	return salaries, nil
}

func resolveGlintsSkills(cache map[string]json.RawMessage, relations []glintsDetailSkillRelation) ([]glintsSkillBasic, error) {
	skills := make([]glintsSkillBasic, 0, len(relations))
	for _, relation := range relations {
		if relation.Skill.Ref == "" {
			continue
		}

		var skill glintsSkillBasic
		if err := unmarshalApolloEntity(cache, relation.Skill.Ref, &skill); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func unmarshalApolloEntity(cache map[string]json.RawMessage, key string, target any) error {
	raw, ok := cache[key]
	if !ok {
		return fmt.Errorf("apollo entity %q not found", key)
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode entity %q: %w", key, err)
	}

	return nil
}

func buildGlintsDetailURL(title, id string) string {
	return glintsDetailBaseURL + glintsSlugFromTitle(title) + "/" + strings.TrimSpace(id)
}

func glintsSlugFromTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = strings.ReplaceAll(title, "&", " and ")
	title = strings.ReplaceAll(title, "+", " plus ")

	var slugBuilder strings.Builder
	lastHyphen := false
	for _, char := range title {
		switch {
		case char >= 'a' && char <= 'z':
			slugBuilder.WriteRune(char)
			lastHyphen = false
		case char >= '0' && char <= '9':
			slugBuilder.WriteRune(char)
			lastHyphen = false
		default:
			if !lastHyphen {
				slugBuilder.WriteRune('-')
				lastHyphen = true
			}
		}
	}

	slug := strings.Trim(slugBuilder.String(), "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	return slug
}

func deriveGlintsCategory(category *glintsListJobCategory) string {
	if category == nil {
		return ""
	}

	return firstNonEmpty(category.Name)
}

func deriveGlintsLocation(location *glintsListLocation) string {
	if location == nil {
		return ""
	}

	parts := make([]string, 0, 1+len(location.Parents))
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(parts, value) {
			return
		}
		parts = append(parts, value)
	}

	appendPart(location.FormattedName)
	appendPart(location.Name)
	for _, parent := range location.Parents {
		appendPart(parent.FormattedName)
	}

	return strings.Join(parts, ", ")
}

func deriveGlintsSalary(shouldShow bool, salaries []glintsListSalary) (*int64, *int64, string) {
	if !shouldShow || len(salaries) == 0 {
		return nil, nil, ""
	}

	var minValue *int64
	var maxValue *int64
	currency := ""

	for _, salary := range salaries {
		if minValue == nil && salary.MinAmount != nil {
			minValue = salary.MinAmount
		}
		if maxValue == nil && salary.MaxAmount != nil {
			maxValue = salary.MaxAmount
		}
		if currency == "" {
			currency = strings.TrimSpace(salary.Currency)
		}
	}

	return minValue, maxValue, currency
}

func deriveGlintsStructuredSalary(posting glintsStructuredJobPosting) (*int64, *int64, string) {
	return posting.BaseSalary.Value.MinValue, posting.BaseSalary.Value.MaxValue, strings.TrimSpace(posting.BaseSalary.Currency)
}

func renderGlintsDraftText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var document glintsDraftJSDocument
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return raw
	}

	lines := make([]string, 0, len(document.Blocks))
	for _, block := range document.Blocks {
		text := strings.TrimSpace(block.Text)
		if text != "" {
			lines = append(lines, text)
		}
	}

	return strings.Join(lines, "\n")
}

func buildGlintsRequirements(job glintsDetailJob, skills []glintsSkillBasic, posting glintsStructuredJobPosting) string {
	lines := make([]string, 0)

	if education := firstNonEmpty(humanizeGlintsEducationLevel(job.EducationLevel), humanizeGlintsCredential(posting.EducationRequirements.CredentialCategory)); education != "" {
		lines = append(lines, "Education: "+education)
	}

	if years := humanizeGlintsExperience(job.MinYearsOfExperience, job.MaxYearsOfExperience); years != "" {
		lines = append(lines, "Experience: "+years)
	}

	skillNames := make([]string, 0, len(skills))
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		skillNames = append(skillNames, name)
	}
	if len(skillNames) > 0 {
		lines = append(lines, "Skills: "+strings.Join(skillNames, ", "))
	}

	return strings.Join(lines, "\n")
}

func normalizeGlintsBenefits(raw any) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case []any:
		lines := make([]string, 0, len(value))
		for _, item := range value {
			switch benefit := item.(type) {
			case string:
				benefit = strings.TrimSpace(benefit)
				if benefit != "" {
					lines = append(lines, benefit)
				}
			case map[string]any:
				name := firstNonEmpty(stringFromAny(benefit["name"]), stringFromAny(benefit["label"]))
				if name != "" {
					lines = append(lines, name)
				}
			}
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func marshalGlintsRawJSON(item glintsListJob, detail glintsDetailData) (string, error) {
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

func humanizeGlintsEducationLevel(value string) string {
	switch strings.TrimSpace(value) {
	case "HIGH_SCHOOL":
		return "High school"
	case "DIPLOMA_DEGREE":
		return "Diploma degree"
	case "BACHELOR_DEGREE":
		return "Bachelor's degree"
	case "MASTER_DEGREE":
		return "Master's degree"
	case "DOCTORAL_DEGREE":
		return "Doctoral degree"
	default:
		return humanizeGlintsEnum(value)
	}
}

func humanizeGlintsExperience(minYears, maxYears *int) string {
	switch {
	case minYears != nil && maxYears != nil && *minYears == *maxYears:
		return fmt.Sprintf("%d year%s", *minYears, pluralizeInt(*minYears))
	case minYears != nil && maxYears != nil:
		return fmt.Sprintf("%d-%d years", *minYears, *maxYears)
	case minYears != nil:
		return fmt.Sprintf("Minimum %d year%s", *minYears, pluralizeInt(*minYears))
	case maxYears != nil:
		return fmt.Sprintf("Up to %d year%s", *maxYears, pluralizeInt(*maxYears))
	default:
		return ""
	}
}

func pluralizeInt(value int) string {
	if value == 1 {
		return ""
	}
	return "s"
}

func humanizeGlintsEnum(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, " ")
}

func humanizeGlintsCredential(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "high school":
		return "High school"
	case "bachelor's degree":
		return "Bachelor's degree"
	case "diploma":
		return "Diploma degree"
	default:
		return humanizeGlintsEnum(strings.ReplaceAll(value, " ", "_"))
	}
}

func deriveGlintsStructuredLocation(posting glintsStructuredJobPosting) string {
	parts := make([]string, 0, 4)
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(parts, value) {
			return
		}
		parts = append(parts, value)
	}

	appendPart(posting.JobLocation.Address.StreetAddress)
	appendPart(posting.JobLocation.Address.AddressLocality)
	appendPart(posting.JobLocation.Address.AddressRegion)
	appendPart(posting.JobLocation.Address.AddressCountry)

	return strings.Join(parts, ", ")
}

func extractGlintsStructuredSections(fragment string) glintsStructuredSections {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return glintsStructuredSections{}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + fragment + "</div>"))
	if err != nil {
		return glintsStructuredSections{Description: htmlToText(fragment)}
	}

	sections := map[string][]string{
		"requirements": {},
		"description":  {},
		"benefits":     {},
	}
	current := "description"

	doc.Find("p, li").Each(func(_ int, selection *goquery.Selection) {
		text := strings.TrimSpace(selection.Text())
		if text == "" {
			return
		}

		lower := strings.ToLower(strings.TrimSuffix(text, ":"))
		switch lower {
		case "kualifikasi", "qualification", "qualifications", "requirements":
			current = "requirements"
			return
		case "job description", "deskripsi pekerjaan", "description":
			current = "description"
			return
		case "benefit", "benefits":
			current = "benefits"
			return
		}

		sections[current] = append(sections[current], text)
	})

	return glintsStructuredSections{
		Requirements: strings.Join(sections["requirements"], "\n"),
		Description:  strings.Join(sections["description"], "\n"),
		Benefits:     strings.Join(sections["benefits"], "\n"),
	}
}

func stringFromAny(value any) string {
	text, _ := value.(string)
	return text
}
