package output

import "regexp"

var redactionPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)\b(token|access[_-]?token|refresh[_-]?token|api[_-]?key|x[_-]?api[_-]?key|password|passwd|secret|client[_-]?secret|credential)\s*([=:])\s*([^\s,;]+)`), `$1$2[REDACTED]`},
	{regexp.MustCompile(`(?i)\b(authorization)\s*:\s*Bearer\s+([^\s,;]+)`), `$1: Bearer [REDACTED]`},
}

// RedactString masks obvious secret values in diagnostic strings.
func RedactString(value string) string {
	for _, rule := range redactionPatterns {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	return value
}
