package helper

func MapGet[T any](data map[string]interface{}, key string) T {
	var resVal T
	val, ok := data[key]
	if ok {
		resVal = val.(T)
	}

	return resVal
}

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	cp := make(map[K]V, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
