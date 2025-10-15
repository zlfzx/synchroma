package config

import (
	"fmt"
	"os"
	"strings"
	"synchroma/internal/models"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func GetConfig(cmd *cobra.Command) (sourceCfg, targetCfg models.DataSource) {
	// load config from flags
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

	if database != "" || sourceDbHost != "" || sourceDbPort != "" || sourceDbUser != "" || sourceDbPass != "" || sourceDbName != "" || targetDbHost != "" || targetDbPort != "" || targetDbUser != "" || targetDbPass != "" || targetDbName != "" {
		sourceCfg = models.DataSource{
			Database: database,
			Host:     sourceDbHost,
			Port:     sourceDbPort,
			User:     sourceDbUser,
			Password: sourceDbPass,
			DBName:   sourceDbName,
		}

		targetCfg = models.DataSource{
			Database: database,
			Host:     targetDbHost,
			Port:     targetDbPort,
			User:     targetDbUser,
			Password: targetDbPass,
			DBName:   targetDbName,
		}

		return
	}

	// home directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// check if init flag is set
	initConfig, _ := cmd.Flags().GetBool("init")
	if initConfig {
		sourceCfg, targetCfg = InputConfig(home)
		return
	}

	// load config from .synchroma file
	err = godotenv.Load(home + "/.synchroma")
	if err != nil {
		// input config if .synchroma file not found
		sourceCfg, targetCfg = InputConfig(home)
		return
	}

	sourceCfg = models.DataSource{
		Database: os.Getenv("DATABASE"),
		Host:     os.Getenv("SOURCE_DB_HOST"),
		Port:     os.Getenv("SOURCE_DB_PORT"),
		User:     os.Getenv("SOURCE_DB_USER"),
		Password: os.Getenv("SOURCE_DB_PASSWORD"),
		DBName:   os.Getenv("SOURCE_DB_NAME"),
	}

	targetCfg = models.DataSource{
		Database: os.Getenv("DATABASE"),
		Host:     os.Getenv("TARGET_DB_HOST"),
		Port:     os.Getenv("TARGET_DB_PORT"),
		User:     os.Getenv("TARGET_DB_USER"),
		Password: os.Getenv("TARGET_DB_PASSWORD"),
		DBName:   os.Getenv("TARGET_DB_NAME"),
	}

	return
}

func InputConfig(homeDir string) (sourceCfg, targetCfg models.DataSource) {
	var database,
		sourceDbHost,
		sourceDbPort,
		sourceDbUser,
		sourceDbPass,
		sourceDbName,
		targetDbHost,
		targetDbPort,
		targetDbUser,
		targetDbPass,
		targetDbName,
		saveConfig string

	fmt.Print("Please provide the database type (mysql): ")
	fmt.Scan(&database)

	fmt.Println("Please provide the source database connection details")

	fmt.Print("- host: ")
	fmt.Scan(&sourceDbHost)

	fmt.Print("- port: ")
	fmt.Scan(&sourceDbPort)

	fmt.Print("- user: ")
	fmt.Scan(&sourceDbUser)

	fmt.Print("- password: ")
	fmt.Scan(&sourceDbPass)

	fmt.Print("- database name: ")
	fmt.Scan(&sourceDbName)

	fmt.Println("Please provide the target database connection details")

	fmt.Print("- host: ")
	fmt.Scan(&targetDbHost)

	fmt.Print("- port: ")
	fmt.Scan(&targetDbPort)

	fmt.Print("- user: ")
	fmt.Scan(&targetDbUser)

	fmt.Print("- password: ")
	fmt.Scan(&targetDbPass)

	fmt.Print("- database name: ")
	fmt.Scan(&targetDbName)

	fmt.Println()

	fmt.Print("Do you want to save this configuration? (y/N): ")
	fmt.Scan(&saveConfig)

	sourceCfg = models.DataSource{
		Database: database,
		Host:     sourceDbHost,
		Port:     sourceDbPort,
		User:     sourceDbUser,
		Password: sourceDbPass,
		DBName:   sourceDbName,
	}

	targetCfg = models.DataSource{
		Database: database,
		Host:     targetDbHost,
		Port:     targetDbPort,
		User:     targetDbUser,
		Password: targetDbPass,
		DBName:   targetDbName,
	}

	if strings.ToLower(saveConfig) != "y" {
		return
	}

	writeConfig := map[string]string{
		"DATABASE":           sourceCfg.Database,
		"SOURCE_DB_HOST":     sourceCfg.Host,
		"SOURCE_DB_PORT":     sourceCfg.Port,
		"SOURCE_DB_USER":     sourceCfg.User,
		"SOURCE_DB_PASSWORD": sourceCfg.Password,
		"SOURCE_DB_NAME":     sourceCfg.DBName,
		"TARGET_DB_HOST":     targetCfg.Host,
		"TARGET_DB_PORT":     targetCfg.Port,
		"TARGET_DB_USER":     targetCfg.User,
		"TARGET_DB_PASSWORD": targetCfg.Password,
		"TARGET_DB_NAME":     targetCfg.DBName,
	}

	env, err := godotenv.Marshal(writeConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	writeConfig, err = godotenv.Unmarshal(env)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	godotenv.Write(writeConfig, homeDir+"/.synchroma")

	fmt.Println("Configuration saved to ~/.synchroma")
	fmt.Println()

	return
}
