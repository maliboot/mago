package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	httpdo "tests/internal/user/infrastructure/do/http"
)

const (
	// defaultWeatherEndpoint 是 open-meteo 的实时天气接口，免密钥。
	defaultWeatherEndpoint = "https://api.open-meteo.com/v1/forecast"
	// defaultWeatherFields 是未指定字段时查询的实时指标。
	defaultWeatherFields = "temperature_2m,relative_humidity_2m,wind_speed_10m,weather_code"
	// weatherMaxBody 限制响应体大小，避免异常响应打爆内存。
	weatherMaxBody = 1 << 20
)

// Weather 是天气查询的 HTTP 出站适配器：请求内容来自 do/http，短连接与错误归一都在本文件。
type Weather struct {
	endpoint string
	client   *http.Client
}

// NewWeather 创建适配器；endpoint 为空时用 open-meteo 默认地址，超时由 client 控制。
// 非单例：endpoint 可按环境切换；持有 http.Client，由组合根创建一次即可复用连接池。
func NewWeather(endpoint string) *Weather {
	if endpoint == "" {
		endpoint = defaultWeatherEndpoint
	}

	return &Weather{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

// Current 查询指定经纬度的实时天气。
func (w *Weather) Current(ctx context.Context, query *httpdo.CurrentWeatherQuery) (*httpdo.CurrentWeatherResult, error) {
	if query == nil {
		return nil, fmt.Errorf("weather: 查询参数为空")
	}
	if query.Latitude < -90 || query.Latitude > 90 || query.Longitude < -180 || query.Longitude > 180 {
		return nil, fmt.Errorf("weather: 经纬度非法 (%v, %v)", query.Latitude, query.Longitude)
	}

	fields := defaultWeatherFields
	if len(query.Fields) > 0 {
		fields = strings.Join(query.Fields, ",")
	}

	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(query.Latitude, 'f', 4, 64))
	params.Set("longitude", strconv.FormatFloat(query.Longitude, 'f', 4, 64))
	params.Set("current", fields)
	if query.Timezone != "" {
		params.Set("timezone", query.Timezone)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("weather: 构造请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather: 请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, weatherMaxBody))
	if err != nil {
		return nil, fmt.Errorf("weather: 读取响应失败: %w", err)
	}

	// 第三方状态码在这里归一，上层不解析原始报文
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather: 服务返回 %d: %s", resp.StatusCode, snippet(body))
	}

	var out httpdo.CurrentWeatherResult
	if err = json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("weather: 解析响应失败: %w", err)
	}

	return &out, nil
}

// snippet 截断错误响应，避免把整段 HTML 写进日志。
func snippet(body []byte) string {
	const max = 200
	if len(body) <= max {
		return string(body)
	}

	return string(body[:max]) + "..."
}
