package micro

func R[T interface{}](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func E(err error) {
	if err != nil {
		panic(err)
	}
}
