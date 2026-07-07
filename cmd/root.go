package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"synchroma/pkg/config"
	"synchroma/pkg/core"
	"synchroma/pkg/models"
	"synchroma/pkg/schema"
	"time"

	"github.com/adhocore/chin"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "synchroma",
	Short: "Synchroma is a tool to synchronize database schemas",
	Long: `Synchroma is a tool to synchronize database schemas. 
It supports MySQL and other databases in the future.

Example:
  synchroma --init
  synchroma --profile staging
  synchroma --dry-run
  synchroma --apply`,
	Run: runCLI,
}

func init() {
	rootCmd.Flags().String("database", "", "Database type (mysql, etc)")
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
	rootCmd.Flags().StringP("profile", "p", "default", "Configuration profile to use")
	rootCmd.Flags().Bool("dry-run", false, "Print SQL to stdout without saving")
	rootCmd.Flags().Bool("apply", false, "Execute the generated SQL directly on the target database")
	rootCmd.Flags().Bool("drop-tables", false, "Drop tables in target that do not exist in source")
	rootCmd.Flags().String("output-file", "", "Custom output SQL filename")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runCLI(cmd *cobra.Command, args []string) {
	start := time.Now()

	initFlag, _ := cmd.Flags().GetBool("init")
	home, _ := os.UserHomeDir()
	configPath := home + "/.synchroma.json"

	if initFlag {
		interactiveConfig(configPath)
		return
	}

	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		profileName = "default"
	}

	sourceCfg, targetCfg, err := loadConfigFromFlagsOrFile(cmd, configPath, profileName)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("Connected to source database:")
	fmt.Println(" - Host:", sourceCfg.Host)
	fmt.Println(" - Database:", sourceCfg.DBName)
	fmt.Println(" ")
	fmt.Println("Connected to target database:")
	fmt.Println(" - Host:", targetCfg.Host)
	fmt.Println(" - Database:", targetCfg.DBName)
	fmt.Println(" ")

	dropTables, _ := cmd.Flags().GetBool("drop-tables")

	var wgChin sync.WaitGroup
	spinner := chin.New().WithWait(&wgChin)
	go spinner.Start()

	opts := core.SyncOptions{
		SourceCfg:  sourceCfg,
		TargetCfg:  targetCfg,
		DropTables: dropTables,
		OnProgress: func(msg string) {
			// Wipe line cleanly for the spinner if needed or just print.
			// Chin might mess with standard prints, so we could pause it or just print directly.
			fmt.Println(msg)
		},
	}

	result, err := core.GenerateSyncSQL(opts)

	spinner.Stop()
	wgChin.Wait()

	if err != nil {
		log.Fatalf("\nSync failed: %v", err)
	}

	printSummary(result.Stats, time.Since(start))

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	apply, _ := cmd.Flags().GetBool("apply")

	if dryRun {
		fmt.Println("\n--- DRY RUN OUTPUT ---")
		fmt.Println(result.SQL)
		return
	}

	if apply {
		fmt.Println("\nApplying SQL to target database...")
		
		targetSchema, err := schema.InitSchema(targetCfg)
		if err != nil {
			log.Fatalf("Failed to connect to target for apply mode: %v", err)
		}
		defer targetSchema.Close()

		if mysqlSchema, ok := targetSchema.(*schema.MySQLSchema); ok {
			_, err := mysqlSchema.DB.Exec(result.SQL)
			if err != nil {
				log.Fatalf("Failed to apply SQL: %v", err)
			}
			fmt.Println("SQL successfully applied to target database!")
		} else {
			log.Fatalf("Apply mode is currently only supported for MySQL target")
		}
		return
	}

	filename, _ := cmd.Flags().GetString("output-file")
	if filename == "" {
		filename = sourceCfg.DBName + "_to_" + targetCfg.DBName + ".sql"
	}
	wd, _ := os.Getwd()
	pathFile := wd + "/" + filename

	f, err := os.Create(pathFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	if _, err = f.WriteString(result.SQL); err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
	fmt.Println("\nSuccess! SQL file has been generated at " + pathFile)
}

func loadConfigFromFlagsOrFile(cmd *cobra.Command, configPath, profileName string) (models.DataSource, models.DataSource, error) {
	database, _ := cmd.Flags().GetString("database")
	sourceDbHost, _ := cmd.Flags().GetString("source-db-host")
	sourceDbPort, _ := cmd.Flags().GetString("source-db-port")
	sourceDbUser, _ := cmd.Flags().GetString("source-db-user")
	sourceDbPass, _ := cmd.Flags().GetString("source-db-password")
	sourceDbName, _ := cmd.Flags().GetString("source-db-name")

	targetDbHost, _ := cmd.Flags().GetString("target-db-host")
	targetDbPort, _ := cmd.Flags().GetString("target-db-port")
	targetDbUser, _ := cmd.Flags().GetString("target-db-user")
	targetDbPass, _ := cmd.Flags().GetString("target-db-password")
	targetDbName, _ := cmd.Flags().GetString("target-db-name")

	if sourceDbHost != "" && targetDbHost != "" {
		src := models.DataSource{
			Database: database, Host: sourceDbHost, Port: sourceDbPort, User: sourceDbUser, Password: sourceDbPass, DBName: sourceDbName,
		}
		tgt := models.DataSource{
			Database: database, Host: targetDbHost, Port: targetDbPort, User: targetDbUser, Password: targetDbPass, DBName: targetDbName,
		}
		return src, tgt, nil
	}

	return config.LoadConfig(configPath, profileName)
}

func interactiveConfig(configPath string) {
	var profileName, database, sHost, sPort, sUser, sPass, tHost, tPort, tUser, tPass, sDBName, tDBName, saveConfig string

	fmt.Print("Please provide profile name (default): ")
	fmt.Scanln(&profileName)
	if profileName == "" {
		profileName = "default"
	}

	fmt.Print("Please provide the database type (mysql): ")
	fmt.Scanln(&database)
	if database == "" {
		database = "mysql"
	}

	fmt.Println("Please provide the source database connection details")
	fmt.Print("- host: ")
	fmt.Scanln(&sHost)
	fmt.Print("- port: ")
	fmt.Scanln(&sPort)
	fmt.Print("- user: ")
	fmt.Scanln(&sUser)
	fmt.Print("- password: ")
	fmt.Scanln(&sPass)
	fmt.Print("- database name: ")
	fmt.Scanln(&sDBName)

	fmt.Println("Please provide the target database connection details")
	fmt.Print("- host: ")
	fmt.Scanln(&tHost)
	fmt.Print("- port: ")
	fmt.Scanln(&tPort)
	fmt.Print("- user: ")
	fmt.Scanln(&tUser)
	fmt.Print("- password: ")
	fmt.Scanln(&tPass)
	fmt.Print("- database name: ")
	fmt.Scanln(&tDBName)

	fmt.Println()
	fmt.Print("Do you want to save this configuration? (y/N): ")
	fmt.Scanln(&saveConfig)

	if strings.ToLower(saveConfig) == "y" {
		src := models.DataSource{Database: database, Host: sHost, Port: sPort, User: sUser, Password: sPass, DBName: sDBName}
		tgt := models.DataSource{Database: database, Host: tHost, Port: tPort, User: tUser, Password: tPass, DBName: tDBName}
		
		err := config.SaveConfig(configPath, profileName, src, tgt)
		if err != nil {
			fmt.Println("Failed to save config:", err)
		} else {
			fmt.Printf("Configuration saved to %s under profile '%s'\n", configPath, profileName)
		}
	}
}

func printSummary(stats *core.SyncStats, d time.Duration) {
	fmt.Println("\n================ SYNCHROMA SUMMARY ================")
	fmt.Printf("Tables  : %d added | %d modified | %d dropped | %d props updated\n", stats.TablesAdded, stats.TablesModified, stats.TablesDropped, stats.TablePropsSynced)
	fmt.Printf("Columns : %d added | %d modified | %d dropped\n", stats.ColumnsAdded, stats.ColumnsModified, stats.ColumnsDropped)
	fmt.Printf("Indexes : %d added | %d dropped\n", stats.IndexesAdded, stats.IndexesDropped)
	fmt.Printf("FKs     : %d added | %d dropped\n", stats.FKsAdded, stats.FKsDropped)
	fmt.Printf("Objects : %d views | %d triggers | %d routines synced\n", stats.ViewsSynced, stats.TriggersSynced, stats.RoutinesSynced)
	fmt.Printf("Time    : %s\n", d)
	fmt.Println("===================================================")
}
