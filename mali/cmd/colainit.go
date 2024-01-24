package cmd

import (
	"fmt"

	"github.com/maliboot/mago/mali/cmd/mod"
	"github.com/maliboot/mago/mali/cmd/tpl"
	"github.com/spf13/cobra"
)

// colaCmd represents the cola command
var colaCmd = &cobra.Command{
	Use:   "init",
	Short: "maliboot-cola骨架",
	Long:  "生成maliboot-cola骨架相关代码。命令示例：mali init -m=./foo",
	Run: func(cmd *cobra.Command, args []string) {
		runCola(moduleDir, force)
	},
}

func init() {
	rootCmd.AddCommand(colaCmd)
	colaCmd.Flags().StringVarP(&moduleDir, "module-path", "m", ".", "模块路径，模块代码将生成在当前目录下")
	colaCmd.Flags().BoolVarP(&force, "force", "f", false, "是否强制覆盖文件")
}

func runCola(modDir string, f bool) {
	var modIns = mod.NewMod(modDir)
	tplExecutors := tpl.GetColaExecutors(modIns, f)
	for i := 0; i < len(tplExecutors); i++ {
		tplExecutors[i].Initialize()
		err := tplExecutors[i].Execute()
		if err != nil {
			cobra.CompErrorln(fmt.Sprintf("%s模板处理器异常：%s", tplExecutors[i].Name(), err.Error()))
		}
	}
	fmt.Println("done.")
}
