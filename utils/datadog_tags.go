package utils

import "strings"

// ExtractTagKey returns the key portion of a "key:value" tag string.
// For tags without a colon, the entire string is treated as the key.
func ExtractTagKey(tag string) string {
	parts := strings.SplitN(tag, ":", 2)
	return parts[0]
}

// MergeTagSlices merges incoming tags into existing tags using local-priority rules.
// For each incoming tag, if an existing tag shares the same key the existing tag
// is kept and the incoming one is discarded. Incoming tags whose key does not
// appear in existing are appended.
func MergeTagSlices(existing, incoming []string) []string {
	if len(incoming) == 0 {
		return existing
	}

	// Index existing tags by key for fast lookup.
	existingByKey := make(map[string]bool, len(existing))
	for _, tag := range existing {
		existingByKey[ExtractTagKey(tag)] = true
	}

	merged := make([]string, len(existing))
	copy(merged, existing)

	// Append only incoming tags whose key is not already in existing.
	for _, tag := range incoming {
		if !existingByKey[ExtractTagKey(tag)] {
			merged = append(merged, tag)
		}
	}

	return merged
}
