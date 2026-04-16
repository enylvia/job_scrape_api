package logger

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
)

func ColorScope(scope string) string {
	return colorCyan + "[" + scope + "]" + colorReset
}

func ColorStart(label string) string {
	return colorMagenta + label + colorReset
}

func ColorFetch(label string) string {
	return colorBlue + label + colorReset
}

func ColorSuccess(label string) string {
	return colorGreen + label + colorReset
}

func ColorWarn(label string) string {
	return colorYellow + label + colorReset
}

func ColorError(label string) string {
	return colorRed + label + colorReset
}
