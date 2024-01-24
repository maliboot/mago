package cmd

import (
	"fmt"

	"github.com/maliboot/mago/mali/cmd/mbast"
	"github.com/maliboot/mago/mali/cmd/mod"
	"github.com/maliboot/mago/mali/cmd/tpl"
	"github.com/spf13/cobra"
)

// injectCmd represents the inject command
var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "依赖注入",
	Long:  "依托google/wire，生成依赖注入代码。命令示例：mali inject -m=./foo",
	Run: func(cmd *cobra.Command, args []string) {
		runInject(moduleDir)
	},
}

func init() {
	rootCmd.AddCommand(injectCmd)
	injectCmd.Flags().StringVarP(&moduleDir, "module-path", "m", ".", "模块路径，wire_gen.go将生成在当前目录下")
}

func runInject(modDir string) {
	var modIns = mod.NewMod(modDir)
	files, err := mbast.NewFiles(modIns)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return
	}
	nodes, err := files.Parser()
	if err != nil {
		cobra.CompErrorln(err.Error())
		return
	}

	tplExecutors := tpl.GetInjectExecutors(modIns, nodes)
	for i := 0; i < len(tplExecutors); i++ {
		tplExecutors[i].Initialize()
		err := tplExecutors[i].Execute()
		if err != nil {
			cobra.CompErrorln(fmt.Sprintf("%s模板处理器异常：%s", tplExecutors[i].Name(), err.Error()))
		}
	}
	fmt.Println("done.")
}
