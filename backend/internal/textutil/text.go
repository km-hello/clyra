package textutil

import "unicode/utf16"

func Length(s string) int { return len(utf16.Encode([]rune(s))) }
func Truncate(value string, limit int) string {
	size := 0
	for i, r := range value {
		n := 1
		if r > 0xffff {
			n = 2
		}
		if size+n > limit {
			return value[:i]
		}
		size += n
	}
	return value
}
