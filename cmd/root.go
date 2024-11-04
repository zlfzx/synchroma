package cmd

import (
	"fmt"
	"os"
	"synchroma/internal/handlers"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "synchroma",
	Short: "Synchroma is a tool to synchronize database schema",
	Long:  "Synchroma is a tool to compare and synchronize schema between two databases.",
	Example: `  synchroma --init
  synchroma \
    --database=mysql \
    --source-db-host=source_host \
    --source-db-port=source_port \
    --source-db-user=source_user \
    --source-db-password=source_password \
    --source-db-name=source_db \
    --target-db-host=target_host \
    --target-db-port=target_port \
    --target-db-user=target_user \
    --target-db-password=target_password \
    --target-db-name=target_db
	`,
	Run: handlers.SyncSchema,
}

func init() {
	rootCmd.Flags().String("database", "", "Database type (mysql)")

	rootCmd.Flags().String("source-db-host", "", "Source database host")
	rootCmd.Flags().String("source-db-port", "", "Source database port")
	rootCmd.Flags().String("source-db-user", "", "Source database user")
	rootCmd.Flags().String("source-db-password", "", "Source database password")
	rootCmd.Flags().String("source-db-name", "", "Source database name")

	rootCmd.Flags().String("target-db-host", "", "Target database host")
	rootCmd.Flags().String("target-db-port", "", "Target database port")
	rootCmd.Flags().String("target-db-user", "", "Target database user")
	rootCmd.Flags().String("target-db-password", "", "Target database password")
	rootCmd.Flags().String("target-db-name", "", "Target database name")

	rootCmd.Flags().BoolP("init", "i", false, "Initialize configuration file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
