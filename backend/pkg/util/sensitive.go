package util

import "regexp"

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|secret|token|credential|database\.url|db\.url)`)

func LooksSensitive(key string) bool {
	return sensitiveKeyPattern.MatchString(key)
}
