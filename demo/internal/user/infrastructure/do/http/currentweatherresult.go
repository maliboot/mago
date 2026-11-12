package http

// CurrentWeatherResult 是「查询实时天气」的响应报文，字段与 open-meteo 的返回对齐。
// 只放结构，不做单位换算与业务判断，换算在调用方或 convertor 里完成。
type CurrentWeatherResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Current   struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		Humidity    float64 `json:"relative_humidity_2m"`
		WindSpeed   float64 `json:"wind_speed_10m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
	CurrentUnits struct {
		Temperature string `json:"temperature_2m"`
		Humidity    string `json:"relative_humidity_2m"`
		WindSpeed   string `json:"wind_speed_10m"`
	} `json:"current_units"`
}
