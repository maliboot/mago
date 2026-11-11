package util

// BoolToInt8 把布尔值映射为数据库 tinyint 的 0/1。
func BoolToInt8(v bool) int8 {
	if v {
		return 1
	}

	return 0
}

// Int8ToBool 把数据库 tinyint 的 0/1 还原为布尔值。
func Int8ToBool(v int8) bool {
	return v != 0
}
