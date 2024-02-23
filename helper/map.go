package helper

func MapGet[T any](data map[string]interface{}, key string) T {
	var resVal T
	val, ok := data[key]
	if ok {
		resVal = val.(T)
	}

	return resVal
}
