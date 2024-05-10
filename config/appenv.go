package config

import "strings"

type AppEnv string

const (
	// Dev 开发环境
	Dev = "dev"
	// Test 测试环境
	Test = "test"
	// Gray 灰度环境
	Gray = "gray"
	// Production 正式环境
	Production = "production"
)

func (e AppEnv) String() string {
	switch e {
	case Dev:
		return "dev"
	case Test:
		return "test"
	case Gray:
		return "gray"
	}
	return "production"
}

func AppEnvFromStr(e string) AppEnv {
	appEnv := strings.ToLower(e)
	switch appEnv {
	case "dev":
		return Dev
	case "test":
		return Test
	case "gray":
		return Gray
	}

	return Production
}
