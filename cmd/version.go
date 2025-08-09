package cmd

import (
	"github.com/krau/remdit/config"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Remdit CLI")
		cmd.Println("Version:", config.Version)
		cmd.Println("Commit:", config.Commit)
		cmd.Println("Build Date:", config.BuildDate)
	},
}
