package utils

import "strings"

// ParseValueByPrefix parses the output and returns the value associated with the given prefix.
func ParseValueByPrefix(output string, prefix string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, prefix) {
			value := strings.TrimPrefix(trimmedLine, prefix)
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// RemoveLinesByPrefix removes lines from the output that start with any of the specified prefixes.
func RemoveLinesByPrefix(prefixs []string, output string) string {
	lines := strings.Split(output, "\n")
	var filteredLines []string
	for _, line := range lines {
		for _, prefix := range prefixs {
			if !strings.Contains(line, prefix) {
				filteredLines = append(filteredLines, line)
			}
		}
	}
	output = strings.Join(filteredLines, "\n")
	return output
}
