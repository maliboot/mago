package helper

import (
	"strings"
	"unicode/utf8"
)

func SafeSubBytes(b []byte, maxChars int) []byte {
	s := string(b)
	if utf8.RuneCountInString(s) <= maxChars {
		return b
	}

	var builder strings.Builder
	var count int
	for _, char := range s {
		if count >= maxChars {
			break
		}
		builder.WriteRune(char)
		count++
	}
	return []byte(builder.String())
}
