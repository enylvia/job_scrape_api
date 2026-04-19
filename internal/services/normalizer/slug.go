package normalizer

import (
	"regexp"
	"strings"
)

var repeatedHyphenPattern = regexp.MustCompile(`-+`)

func generateSlug(value string) string {
	value = strings.ToLower(cleanSingleLine(value))
	if value == "" {
		return ""
	}

	value = strings.ReplaceAll(value, "&", " and ")
	value = strings.ReplaceAll(value, "+", " plus ")

	var builder strings.Builder
	lastHyphen := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastHyphen = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastHyphen = false
		default:
			if !lastHyphen {
				builder.WriteRune('-')
				lastHyphen = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	return repeatedHyphenPattern.ReplaceAllString(slug, "-")
}
