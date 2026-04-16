package enums

type JobStatus string

const (
	JobStatusScraped       JobStatus = "scraped"
	JobStatusNormalized    JobStatus = "normalized"
	JobStatusDuplicate     JobStatus = "duplicate"
	JobStatusReviewPending JobStatus = "review_pending"
	JobStatusApproved      JobStatus = "approved"
	JobStatusPublished     JobStatus = "published"
	JobStatusRejected      JobStatus = "rejected"
	JobStatusExpired       JobStatus = "expired"
	JobStatusArchived      JobStatus = "archived"
)

func AllJobStatuses() []JobStatus {
	return []JobStatus{
		JobStatusScraped,
		JobStatusNormalized,
		JobStatusDuplicate,
		JobStatusReviewPending,
		JobStatusApproved,
		JobStatusPublished,
		JobStatusRejected,
		JobStatusExpired,
		JobStatusArchived,
	}
}
