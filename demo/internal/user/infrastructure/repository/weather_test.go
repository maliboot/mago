package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpdo "tests/internal/user/infrastructure/do/http"
)

// openMeteoBody 是 open-meteo 实时天气响应的精简样本。
const openMeteoBody = `{
  "latitude": 39.9042,
  "longitude": 116.4074,
  "timezone": "Asia/Shanghai",
  "current": {"time": "2026-11-11T16:30", "temperature_2m": 12.4, "relative_humidity_2m": 38, "wind_speed_10m": 9.6, "weather_code": 3},
  "current_units": {"temperature_2m": "°C", "relative_humidity_2m": "%", "wind_speed_10m": "km/h"}
}`

// newWeatherServer 起一个假 open-meteo，记录收到的查询参数。
func newWeatherServer(t *testing.T, body string, status int, gotQuery *string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		*gotQuery = req.URL.RawQuery
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestWeatherCurrent(t *testing.T) {
	var gotQuery string
	server := newWeatherServer(t, openMeteoBody, http.StatusOK, &gotQuery)

	weather := NewWeather(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := weather.Current(ctx, &httpdo.CurrentWeatherQuery{
		Latitude:  39.9042,
		Longitude: 116.4074,
		Timezone:  "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}

	// 请求参数必须落在 query 上
	for _, want := range []string{"latitude=39.9042", "longitude=116.4074", "timezone=Asia%2FShanghai", "current="} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("请求缺少参数 %q，实际 %q", want, gotQuery)
		}
	}
	if !strings.Contains(gotQuery, "temperature_2m") || !strings.Contains(gotQuery, "wind_speed_10m") {
		t.Fatalf("默认字段未带上: %q", gotQuery)
	}

	if out.Current.Temperature != 12.4 || out.Current.Humidity != 38 || out.Current.WindSpeed != 9.6 {
		t.Fatalf("解析结果异常: %+v", out.Current)
	}
	if out.CurrentUnits.Temperature != "°C" || out.Timezone != "Asia/Shanghai" {
		t.Fatalf("单位或时区解析异常: %+v", out)
	}
}

func TestWeatherCustomFields(t *testing.T) {
	var gotQuery string
	server := newWeatherServer(t, openMeteoBody, http.StatusOK, &gotQuery)

	weather := NewWeather(server.URL)
	if _, err := weather.Current(context.Background(), &httpdo.CurrentWeatherQuery{
		Latitude: 31.2304, Longitude: 121.4737, Fields: []string{"temperature_2m"},
	}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	if !strings.Contains(gotQuery, "current=temperature_2m") {
		t.Fatalf("自定义字段未生效: %q", gotQuery)
	}
	if strings.Contains(gotQuery, "wind_speed_10m") {
		t.Fatalf("未指定的字段不应出现在请求里: %q", gotQuery)
	}
}

func TestWeatherRejectsBadInput(t *testing.T) {
	weather := NewWeather("http://127.0.0.1:1")

	if _, err := weather.Current(context.Background(), nil); err == nil {
		t.Fatal("nil 参数应报错")
	}
	if _, err := weather.Current(context.Background(), &httpdo.CurrentWeatherQuery{Latitude: 100, Longitude: 0}); err == nil {
		t.Fatal("非法纬度应报错")
	}
}

func TestWeatherServiceError(t *testing.T) {
	var gotQuery string
	server := newWeatherServer(t, `{"reason":"bad request"}`, http.StatusBadRequest, &gotQuery)

	weather := NewWeather(server.URL)
	_, err := weather.Current(context.Background(), &httpdo.CurrentWeatherQuery{Latitude: 39.9, Longitude: 116.4})
	if err == nil {
		t.Fatal("非 200 应报错")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("错误信息应带状态码: %v", err)
	}
}

func TestWeatherBadJSON(t *testing.T) {
	var gotQuery string
	server := newWeatherServer(t, "<html>oops</html>", http.StatusOK, &gotQuery)

	weather := NewWeather(server.URL)
	if _, err := weather.Current(context.Background(), &httpdo.CurrentWeatherQuery{Latitude: 39.9, Longitude: 116.4}); err == nil {
		t.Fatal("响应不是 JSON 时应报错")
	}
}

func TestWeatherContextCanceled(t *testing.T) {
	var gotQuery string
	server := newWeatherServer(t, openMeteoBody, http.StatusOK, &gotQuery)

	weather := NewWeather(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := weather.Current(ctx, &httpdo.CurrentWeatherQuery{Latitude: 39.9, Longitude: 116.4}); err == nil {
		t.Fatal("ctx 取消后应报错")
	}
}
