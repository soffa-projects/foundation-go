package h

import "github.com/thoas/go-funk"

func EmptyIfNull[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}

func ContainsString(array []string, value string) bool {
	return funk.ContainsString(array, value)
}

func ContainsAnyString(array []string, values []string) bool {
	for _, value := range values {
		if ContainsString(array, value) {
			return true
		}
	}
	return false
}
