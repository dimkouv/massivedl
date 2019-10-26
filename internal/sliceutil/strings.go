package sliceutil

import "strings"

// StrIndexOf returns the index of a value in slice of strings or -1 if not found
func StrIndexOf(s []string, v string) int {
	for i := 0; i < len(s); i++ {
		if strings.Compare(v, s[i]) == 0 {
			return i
		}
	}
	return -1
}
