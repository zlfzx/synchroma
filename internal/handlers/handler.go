package handlers

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"synchroma/internal/models"
	"synchroma/internal/services"
	"time"

	"github.com/adhocore/chin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func getConfig(cmd *cobra.Command) (sourceCfg, targetCfg models.DataSource) {
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
		sourceCfg, targetCfg = inputConfig(home)
		return
	}

	// load config from .synchroma file
	err = godotenv.Load(home + "/.synchroma")
	if err != nil {
		// input config if .synchroma file not found
		sourceCfg, targetCfg = inputConfig(home)
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

func inputConfig(homeDir string) (sourceCfg, targetCfg models.DataSource) {
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

func SyncSchema(cmd *cobra.Command, args []string) {
	// init config
	sourceCfg, targetCfg := getConfig(cmd)

	var wg sync.WaitGroup
	s := chin.New().WithWait(&wg)
	go s.Start()

	// init source schema
	sourceSchema := services.InitSchema(sourceCfg)
	defer sourceSchema.DB.Close()
	fmt.Println("Connected to source database:")
	fmt.Println(" - Host:", sourceCfg.Host)
	fmt.Println(" - Database:", sourceCfg.DBName)
	fmt.Println(" ")

	// init target schema
	targetSchema := services.InitSchema(targetCfg)
	defer targetSchema.DB.Close()
	fmt.Println("Connected to target database:")
	fmt.Println(" - Host:", targetCfg.Host)
	fmt.Println(" - Database:", targetCfg.DBName)
	fmt.Println(" ")

	outputSql := outputSQL()

	// check tables
	diffTables := make(map[string]models.Table)

	sourceTables := sourceSchema.GetTables()
	for _, sourceTable := range sourceTables {
		diffTables[sourceTable.TableName.String] = sourceTable
	}

	targetTables := targetSchema.GetTables()
	mapTargetTables := make(map[string]models.Table)
	for _, targetTable := range targetTables {
		delete(diffTables, targetTable.TableName.String)
		mapTargetTables[targetTable.TableName.String] = targetTable
	}

	if len(diffTables) != 0 {
		fmt.Println("Found " + fmt.Sprint(len(diffTables)) + " different tables")

		for _, v := range diffTables {
			fmt.Println(" [✓] Create table:", v.TableName.String)

			output := sourceSchema.CreateTable(v.TableName.String)

			outputSql += output + "\n\n"
		}
	} else {
		fmt.Println("No different tables found")
	}

	fmt.Println(" ")

	// check columns
	for _, sourceTable := range sourceTables {
		if targetTable, ok := mapTargetTables[sourceTable.TableName.String]; ok {
			diffColumns := make(map[string]models.Column)

			sourceColumns := sourceSchema.GetColumns(sourceTable.TableName.String)
			for _, sourceColumn := range sourceColumns {
				diffColumns[sourceColumn.ColumnName.String] = sourceColumn
			}

			targetColumns := targetSchema.GetColumns(targetTable.TableName.String)
			for _, targetColumn := range targetColumns {
				delete(diffColumns, targetColumn.ColumnName.String)
			}

			if len(diffColumns) != 0 {
				output := sourceSchema.CreateColumn(sourceTable.TableName.String, sourceColumns, diffColumns)
				outputSql += output + "\n"

				fmt.Println("Found " + fmt.Sprint(len(diffColumns)) + " different columns for table: " + sourceTable.TableName.String)
				for _, v := range diffColumns {
					fmt.Println(" [✓] Create column:", v.ColumnName.String)
				}
				fmt.Println()
			}
		}
	}

	// filename
	filename := sourceCfg.DBName + "_to_" + targetCfg.DBName + ".sql"

	// get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	pathFile := wd + "/" + filename

	// write sql to file
	f, err := os.Create(pathFile)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer f.Close()

	_, err = f.WriteString(outputSql)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Println("Success!\nSQL file has been generated at " + pathFile)

	s.Stop()
	wg.Wait()
}

func outputSQL() (outputSql string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	outputSql += "-- Generate from Synchroma at " + timestamp + "\n\n"

	return
}
