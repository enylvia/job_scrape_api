package normalizer

import "strings"

func normalizeLocation(value string) string {
	value = cleanSingleLine(strings.ReplaceAll(value, "\n", ", "))
	if value == "" {
		return ""
	}

	parts := strings.Split(value, ",")
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = cleanSingleLine(part)
		if part == "" {
			continue
		}

		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, part)
	}

	return strings.Join(cleaned, ", ")
}
