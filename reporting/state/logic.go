package state

import (
	"fmt"
)

// FormatCount adds commas to large numbers for better readability (e.g., 1000000 -> 1,000,000).
func FormatCount(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var res []byte
	for i, j := len(s)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j > 0 && j%3 == 0 {
			res = append(res, ',')
		}
		res = append(res, s[i])
	}
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

// Reorder moves an element in a slice from one index to another, returning a new slice.
func Reorder[T any](s []T, from, to int) []T {
	if from == to {
		return s
	}
	res := make([]T, len(s))
	copy(res, s)
	val := res[from]
	res = append(res[:from], res[from+1:]...)
	res = append(res[:to], append([]T{val}, res[to:]...)...)
	return res
}
