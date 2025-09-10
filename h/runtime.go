package h

import (
	"reflect"
)

func Safe[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func IsSameFunc(a, b any) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
