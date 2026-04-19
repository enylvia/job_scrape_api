package normalizer

import "strings"

func normalizeEmploymentType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "fulltime", "full_time", "permanent":
		return "full_time"
	case "parttime", "part_time":
		return "part_time"
	case "contract", "contractor", "temporary":
		return "contract"
	case "intern", "internship":
		return "internship"
	case "freelance", "freelancer":
		return "freelance"
	default:
		return normalized
	}
}
