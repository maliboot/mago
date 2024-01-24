package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	moduleDir = "."
	force     = false
)

var rootCmd = &cobra.Command{
	Use:   "mali",
	Short: "MaliBoot框架工具",
	Long:  "生成依赖注入，骨架...",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("execute %s args:%v \n", cmd.Name(), args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
