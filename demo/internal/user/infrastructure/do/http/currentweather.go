// Package http 存放 HTTP 类外部调用的报文（DO），按通用协议分类命名：
// 只描述请求/响应字段，不建连接、不发请求，调用逻辑在 infrastructure/repository/weather.go。
package http

// CurrentWeatherQuery 是「查询实时天气」的请求内容。
type CurrentWeatherQuery struct {
	// Latitude 是纬度（WGS84）。
	Latitude float64
	// Longitude 是经度（WGS84）。
	Longitude float64
	// Timezone 是返回时间所用时区，如 Asia/Shanghai；为空时由服务端默认。
	Timezone string
	// Fields 是需要返回的实时字段，如 temperature_2m、wind_speed_10m。
	Fields []string
}
