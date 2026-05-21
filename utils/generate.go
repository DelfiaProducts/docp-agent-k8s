package utils

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// CompareValuesWithMd5Hash return with values is equals
func CompareValuesWithMd5Hash(current, received []byte) bool {
	curr := md5.Sum(current)
	hashCurrent := hex.EncodeToString(curr[:])
	recei := md5.Sum(received)
	hashReceived := hex.EncodeToString(recei[:])
	return hashCurrent == hashReceived
}

// GenerateHashMd5 generate hash md5
func GenerateHashMd5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// GenerateDatadogHash generates a deterministic hash from content and host tags.
// HostTags are sorted before hashing so order does not affect the result.
func GenerateDatadogHash(content string, hostTags []string) string {
	sorted := make([]string, len(hostTags))
	copy(sorted, hostTags)
	sort.Strings(sorted)
	input := content + "|" + strings.Join(sorted, ",")
	return GenerateHashMd5([]byte(input))
}
