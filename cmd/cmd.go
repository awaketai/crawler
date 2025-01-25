package cmd

import (
	"github.com/awaketai/crawler/cmd/master"
	"github.com/awaketai/crawler/cmd/worker"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "run worker service",
	Long:  "run woker service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		worker.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version",
	Long:  "print version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() error {
	var rootCmd = &cobra.Command{Use: "crawler"}
	rootCmd.AddCommand(master.MasterCmd, workerCmd, versionCmd)
	return rootCmd.Execute()
}
