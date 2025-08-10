package h

func EmptyIfNull[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}
