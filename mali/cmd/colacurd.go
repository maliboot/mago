package cmd

import (
	"fmt"

	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/mali/cmd/mod"
	"github.com/maliboot/mago/mali/cmd/tpl"
	"github.com/spf13/cobra"
)

var databaseTable = ""

// colaCurd represents the cola command
var colaCurd = &cobra.Command{
	Use:   "curd",
	Short: "maliboot-cola curd代码生成",
	Long:  "在maliboot-cola下，生成添加、删除、修改、查询、分页相关代码。命令示例：mali curd -m=./foo",
	Run: func(cmd *cobra.Command, args []string) {
		runColaCurd(moduleDir, databaseTable, force)
	},
}

func init() {
	rootCmd.AddCommand(colaCurd)
	colaCurd.Flags().StringVarP(&moduleDir, "module-path", "m", ".", "模块路径，模块代码将生成在当前目录下")
	colaCurd.Flags().StringVarP(&databaseTable, "table", "t", "", "数据库名称(可选)+表名称，以英文'.'拼接，如-t=goods.attribute。如果不送数据库名称，将会默认用数据库配置的第一个配置")
	colaCurd.Flags().BoolVarP(&force, "force", "f", false, "是否强制覆盖文件")
}

func runColaCurd(modDir string, dbTable string, f bool) {
	if dbTable == "" || dbTable == "." {
		panic("-table不可为空")
	}

	var modIns = mod.NewMod(modDir)
	var configFile = modIns.GetPath() + "/conf.yml"
	c := config.NewConf(config.WithConfFile(configFile))
	if err := c.Bootstrap(); err != nil {
		panic(fmt.Sprintf("数据库配置异常:%v", err))
	}

	tplExecutors := tpl.GetColaCurdExecutors(modIns, c, dbTable, f)
	for i := 0; i < len(tplExecutors); i++ {
		tplExecutors[i].Initialize()
		err := tplExecutors[i].Execute()
		if err != nil {
			cobra.CompErrorln(fmt.Sprintf("%s模板处理器异常：%s", tplExecutors[i].Name(), err.Error()))
		}
	}
	fmt.Println("done.")
}
