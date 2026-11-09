module github.com/maliboot/mago

go 1.26

// 因 beyondstorage.io 兼容性问题 腾讯云cos固定版本号不升级=>github.com/tencentyun/cos-go-sdk-v5 v0.7.32
require (
	github.com/cloudwego/hertz v0.10.5
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/go-co-op/gocron/v2 v2.21.2
	github.com/go-sql-driver/mysql v1.10.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/google/uuid v1.6.0
	github.com/hertz-contrib/http2 v0.1.8
	github.com/hertz-contrib/jwt v1.0.4
	github.com/hertz-contrib/logger/logrus v1.0.1
	github.com/hertz-contrib/logger/slog v1.0.0
	github.com/hibiken/asynq v0.26.0
	github.com/iancoleman/strcase v0.3.0
	github.com/panjf2000/ants/v2 v2.12.1
	github.com/redis/go-redis/v9 v9.21.0
	github.com/shirou/gopsutil/v4 v4.26.5
	github.com/sirupsen/logrus v1.9.4
	github.com/tencentyun/cos-go-sdk-v5 v0.7.32
	go.beyondstorage.io/services/cos/v3 v3.0.0
	go.beyondstorage.io/services/fs/v4 v4.0.0
	go.beyondstorage.io/services/oss/v3 v3.0.0
	go.beyondstorage.io/v5 v5.0.0
	golang.org/x/net v0.56.0
	golang.org/x/sync v0.21.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
	maragu.dev/goqite v0.4.0
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Xuanwo/gg v0.3.0 // indirect
	github.com/Xuanwo/go-bufferpool v0.2.0 // indirect
	github.com/Xuanwo/templateutils v0.2.0 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.2 // indirect
	github.com/bytedance/sonic/loader v0.5.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/cloudwego/gopkg v0.2.0 // indirect
	github.com/cloudwego/netpoll v0.7.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/dave/dst v0.27.4 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/elastic/pkcs8 v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/kevinburke/go-bindata v3.24.0+incompatible // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20260330125221-c963978e514e // indirect
	github.com/mattn/go-sqlite3 v1.14.47 // indirect
	github.com/mozillazg/go-httpheader v0.4.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/qingstor/go-mime v0.1.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/urfave/cli/v2 v2.27.7 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.beyondstorage.io/credential v1.0.0 // indirect
	go.beyondstorage.io/endpoint v1.2.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/arch v0.28.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
