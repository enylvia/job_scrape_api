package normalizer

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlTagPattern       = regexp.MustCompile(`(?s)<[^>]*>`)
	multiWhitespaceRegex = regexp.MustCompile(`[ \t\r\f\v]+`)
)

func cleanSingleLine(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = multiWhitespaceRegex.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func cleanMultiline(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "<br>", "\n")
	value = strings.ReplaceAll(value, "<br/>", "\n")
	value = strings.ReplaceAll(value, "<br />", "\n")
	value = htmlTagPattern.ReplaceAllString(value, " ")

	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = multiWhitespaceRegex.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}
