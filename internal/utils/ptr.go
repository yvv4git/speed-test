package utils

func Ptr[T any](value T) *T {
	return &value
}

func Deref[T any](ptr *T) T {
	return *ptr
}
