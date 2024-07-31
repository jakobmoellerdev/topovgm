package utils

import (
	"regexp"
	"strings"
)

// InLeftButNotInRight returns the difference between two slices, where it is returned what is in a but not in b.
// The order of the returned slice is the same as in a.
// Example:
// a := []string{"a", "b", "c"}
// b := []string{"b", "c", "d"}
// InLeftButNotInRight(a, b) == []string{"a"}
func InLeftButNotInRight[T comparable](left, right []T) []T {
	if len(left) == 0 {
		return nil
	}
	if len(right) == 0 {
		return left
	}

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

// Map maps a slice of type T to a slice of type R using the provided map function.
func Map[T any, R any](original []T, mapf func(T) R) []R {
	slice := make([]R, 0, len(original))
	for _, x := range original {
		slice = append(slice, mapf(x))
	}
	return slice
}

// SequentialTwoWaySync synchronizes two slices of type T as desired and current.
// It calls new with the elements that are in desired but not in current.
// It calls old with the elements that are in current but not in desired.
func SequentialTwoWaySync[T comparable](
	desired []T,
	current []T,
	new func(diff []T) error,
	old func(diff []T) error,
) error {
	var newFromA, oldFromB []T
	newFromA = InLeftButNotInRight(desired, current)
	oldFromB = InLeftButNotInRight(current, desired)

	if len(newFromA) > 0 {
		if err := new(newFromA); err != nil {
			return err
		}
	}

	if len(oldFromB) > 0 {
		if err := old(oldFromB); err != nil {
			return err
		}
	}

	return nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
