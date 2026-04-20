package normalizer

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	htmlTagPattern       = regexp.MustCompile(`(?s)<[^>]*>`)
	multiWhitespaceRegex = regexp.MustCompile(`[ \t\r\f\v]+`)
	bulletPrefixPattern  = regexp.MustCompile(`^\s*([*•-]|\d+\.)\s*`)
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
		line = normalizeMultilineLine(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

func normalizeMultilineLine(value string) string {
	value = bulletPrefixPattern.ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if shouldHumanizeToken(value) {
		return humanizeToken(value)
	}

	return value
}

func shouldHumanizeToken(value string) bool {
	if strings.ContainsAny(value, " \t") {
		return false
	}

	if strings.Contains(value, "_") || strings.Contains(value, "-") {
		return true
	}

	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	for i, r := range value {
		if i == 0 {
			continue
		}
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

func humanizeToken(value string) string {
	if value == "" {
		return ""
	}

	var builder strings.Builder
	runes := []rune(value)
	for index, r := range runes {
		if r == '_' || r == '-' {
			builder.WriteRune(' ')
			continue
		}

		if index > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1])) {
			builder.WriteRune(' ')
		}

		builder.WriteRune(r)
	}

	words := strings.Fields(builder.String())
	for i, word := range words {
		words[i] = capitalizeWord(word)
	}

	return strings.Join(words, " ")
}

func capitalizeWord(word string) string {
	if word == "" {
		return ""
	}

	isShortAllCaps := true
	for _, r := range word {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			isShortAllCaps = false
			break
		}
	}
	if isShortAllCaps && len(word) <= 4 {
		return word
	}

	lower := strings.ToLower(word)
	runes := []rune(lower)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
