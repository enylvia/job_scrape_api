package deduplicator

import (
	"strings"

	"job_aggregator/internal/models"
)

func detectExactDuplicate(candidate, existing models.Job) (bool, string) {
	switch {
	case sameNonEmpty(candidate.SourceJobURL, existing.SourceJobURL):
		return true, "exact duplicate by source_job_url"
	case sameNonEmpty(candidate.SourceApplyURL, existing.SourceApplyURL):
		return true, "exact duplicate by source_apply_url"
	case sameNonEmpty(candidate.ContentHash, existing.ContentHash):
		return true, "exact duplicate by content_hash"
	default:
		return false, ""
	}
}

func sameNonEmpty(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	return left != "" && right != "" && left == right
}
