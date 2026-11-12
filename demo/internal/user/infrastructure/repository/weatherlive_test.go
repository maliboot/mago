//go:build weatherlive

// 真实天气接口联调：只在显式指定 tag 时运行。
//
//	cd demo/internal/user/infrastructure/repository && go test -tags weatherlive -run TestWeatherLive -v -timeout 1m
//
// 访问 open-meteo 公开接口（免密钥），只读、不写任何数据。
package repository

import (
	"context"
	"testing"
	"time"

	httpdo "tests/internal/user/infrastructure/do/http"
)

func TestWeatherLive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	weather := NewWeather("")

	// 北京天安门
	out, err := weather.Current(ctx, &httpdo.CurrentWeatherQuery{
		Latitude:  39.9042,
		Longitude: 116.4074,
		Timezone:  "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}

	if out.Timezone != "Asia/Shanghai" {
		t.Fatalf("时区 = %q", out.Timezone)
	}
	if out.Current.Time == "" {
		t.Fatalf("缺少观测时间: %+v", out.Current)
	}
	if out.CurrentUnits.Temperature == "" {
		t.Fatalf("缺少温度单位: %+v", out.CurrentUnits)
	}

	t.Logf("北京实时天气: %s 温度 %.1f%s 湿度 %.0f%s 风速 %.1f%s 天气码 %d",
		out.Current.Time,
		out.Current.Temperature, out.CurrentUnits.Temperature,
		out.Current.Humidity, out.CurrentUnits.Humidity,
		out.Current.WindSpeed, out.CurrentUnits.WindSpeed,
		out.Current.WeatherCode,
	)
}
