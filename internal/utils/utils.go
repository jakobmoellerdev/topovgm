package utils

// InLeftButNotInRight returns the difference between two slices, where it is returned what is in a but not in b.
// The order of the returned slice is the same as in a.
// Example:
// a := []string{"a", "b", "c"}
// b := []string{"b", "c", "d"}
// InLeftButNotInRight(a, b) == []string{"a"}
func InLeftButNotInRight[T comparable](left, right []T) []T {
	m := make(map[T]struct{}, len(right))
	for _, x := range right {
		m[x] = struct{}{}
	}
	var diff []T
	for _, x := range left {
		if _, ok := m[x]; !ok {
			diff = append(diff, x)
		}
	}
	return diff
}

// ConvertSlice converts a slice of type T to a slice of type R using the provided conversion function.
func ConvertSlice[T any, R any](original []T, convert func(T) R) []R {
	slice := make([]R, 0, len(original))
	for _, x := range original {
		slice = append(slice, convert(x))
	}
	return slice
}
