package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Synchroma",
	Long:  `All software has versions. This is Synchroma's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Synchroma v%s\n", Version)
	},
}
