package helper

func isNil[T comparable](arg T) bool {
	var t T
	return arg == t
}
