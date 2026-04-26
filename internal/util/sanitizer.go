package util

import (
	"regexp"
)

var sensitiveHeaderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^(Authorization):\s+.*$`),
	regexp.MustCompile(`(?im)^(Proxy-Authorization):\s+.*$`),
	regexp.MustCompile(`(?im)^(Cookie):\s+.*$`),
	regexp.MustCompile(`(?im)^(Set-Cookie):\s+.*$`),
	regexp.MustCompile(`(?im)^(X-Api-Key):\s+.*$`),
	regexp.MustCompile(`(?im)^(Api-Key):\s+.*$`),
	regexp.MustCompile(`(?im)^(X-Auth-Token):\s+.*$`),
}

// SanitizeHTTPHeaders replaces sensitive HTTP headers with [REDACTED].
func SanitizeHTTPHeaders(data string) string {
	for _, re := range sensitiveHeaderPatterns {
		data = re.ReplaceAllString(data, "$1: [REDACTED]")
	}
	return data
}
