// 数据迁移脚本：建用户表并补齐默认用户「张三」，可重复执行。
//
//	cd demo && go run ./internal/user/adapter/script/seeduser -f ./conf.yml
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/maliboot/mago/config"

	"tests/internal/user/infrastructure/repository"
)

func main() {
	var confPath string
	flag.StringVar(&confPath, "f", "./conf.yml", "配置文件路径")
	flag.Parse()

	conf := config.NewConf(config.WithConfFile(confPath))
	if err := conf.Bootstrap(false); err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := repository.NewMigrator(conf).Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "数据迁移失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("数据迁移完成：建表=%v 种子用户=%s 本次新建=%v\n", result.TableReady, result.SeedEmail, result.SeedCreated)
}
