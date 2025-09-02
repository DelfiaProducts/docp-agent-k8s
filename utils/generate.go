package utils

import (
	"crypto/md5"
	"encoding/hex"
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
