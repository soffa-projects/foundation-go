package micro

func EmptyListIfNull[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}
