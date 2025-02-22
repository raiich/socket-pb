package must

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func NoError(err error) {
	if err != nil {
		panic(err)
	}
}

func Exists[T any](v T, ok bool) T {
	True(ok)
	return v
}

func True(v bool) {
	if !v {
		panic("unexpected false")
	}
}

func False(v bool) {
	True(!v)
}
