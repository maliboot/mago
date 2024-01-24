package config

type AppEnv int

const (
	// Dev 开发环境
	Dev = 0
	// Test 测试环境
	Test = 1
	// Gray 灰度环境
	Gray = 2
	// Production 正式环境
	Production = 3
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
	switch e {
	case "dev":
		return Dev
	case "test":
		return Test
	case "gray":
		return Gray
	}

	return Production
}
