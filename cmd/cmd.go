package cmd

import (
	"github.com/awaketai/crawler/cmd/master"
	"github.com/awaketai/crawler/cmd/worker"
	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(master.MasterCmd, worker.WorkerCmd, versionCmd)
	return rootCmd.Execute()
}
