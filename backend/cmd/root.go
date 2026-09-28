package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:           "picflow",
	Short:         "PicFlow image processing service",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute 执行 PicFlow 命令行程序。
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(newServeCmd())
}
