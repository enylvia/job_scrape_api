package models

type JobCategoryStat struct {
	Category string `json:"category"`
	JobCount int    `json:"job_count"`
}
