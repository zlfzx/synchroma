package handlers

import (
	"fmt"
	"log"
	"os"
	"sync"
	"synchroma/internal/config"
	"synchroma/internal/models"
	"synchroma/internal/schema"
	"synchroma/internal/utils"
	"time"

	"github.com/adhocore/chin"
	"github.com/spf13/cobra"
)

func SyncSchema(cmd *cobra.Command, args []string) {
	// start time
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("Execution time: %s\n", elapsed)
	}()

	// init config
	sourceCfg, targetCfg := config.GetConfig(cmd)

	var wg sync.WaitGroup
	s := chin.New().WithWait(&wg)
	go s.Start()

	// init source schema
	sourceSchema := schema.InitSchema(sourceCfg)
	defer sourceSchema.DB.Close()
	fmt.Println("Connected to source database:")
	fmt.Println(" - Host:", sourceCfg.Host)
	fmt.Println(" - Database:", sourceCfg.DBName)
	fmt.Println(" ")

	// init target schema
	targetSchema := schema.InitSchema(targetCfg)
	defer targetSchema.DB.Close()
	fmt.Println("Connected to target database:")
	fmt.Println(" - Host:", targetCfg.Host)
	fmt.Println(" - Database:", targetCfg.DBName)
	fmt.Println(" ")

	outputSql := outputSQL(sourceCfg, targetCfg)
	// disable foreign key checks
	outputSql += "SET FOREIGN_KEY_CHECKS=0;\n\n"

	// check tables
	tableDependencies := sourceSchema.GetTableDependencies()
	tables := []string{}
	for _, t := range sourceSchema.GetTables() {
		tables = append(tables, t.TableName.String)
	}

	// build table index to maintain stable order
	tableIndex := make(map[string]int)
	for i, table := range tables {
		tableIndex[table] = i
	}

	// topological sort tables based on foreign key dependencies
	graph := utils.BuildDependencyGraph(tables, tableDependencies)
	orderedTables := utils.TopologicalSort(graph, tableIndex)
	fmt.Println("Processing tables in order to respect foreign key dependencies...")

	targetTables := make(map[string]models.Table)
	for _, targetTable := range targetSchema.GetTables() {
		targetTables[targetTable.TableName.String] = targetTable
	}

	for _, tableName := range orderedTables {
		if _, exists := targetTables[tableName]; !exists {
			// create table if not exists
			outputSql += "\n-- Table: " + tableName + "\n"
			output := sourceSchema.CreateTable(tableName)
			outputSql += output + "\n\n"

			fmt.Println(" [✓] Create table:", tableName)
		} else {
			// compare columns
			outputSql += compareColumns(&sourceSchema, &targetSchema, tableName)

			// compare indexes
			outputSql += compareIndexes(&sourceSchema, &targetSchema, tableName)

			// compare foreign keys
			outputSql += compareForeignKeys(&sourceSchema, &targetSchema, tableName)
		}
	}

	// drop foreign key checks at the end of the sql file
	outputSql += "\nSET FOREIGN_KEY_CHECKS=1;\n"

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

	fmt.Println(" ")
	fmt.Println("Success!\nSQL file has been generated at " + pathFile)

	// cleanup
	sourceSchema.Close()
	targetSchema.Close()

	// stop chin
	s.Stop()
	wg.Wait()
}

func compareColumns(sourceSchema, targetSchema *schema.Schema, tableName string) string {
	outputSql := ""

	sourceColumns := sourceSchema.GetColumns(tableName)
	sourceMapColumns := make(map[string]models.Column)
	for _, sourceColumn := range sourceColumns {
		sourceMapColumns[sourceColumn.ColumnName.String] = sourceColumn
	}

	targetColumns := targetSchema.GetColumns(tableName)
	targetMapColumns := make(map[string]models.Column)
	for _, targetColumn := range targetColumns {
		targetMapColumns[targetColumn.ColumnName.String] = targetColumn
	}

	// compare columns
	for name, col := range sourceMapColumns {
		// add columns
		if _, ok := targetMapColumns[name]; !ok {
			output := sourceSchema.CreateAddColumn(sourceColumns, col)
			outputSql += output + "\n"

			fmt.Printf(" [✓] Add column: %s to table: %s\n", name, tableName)
		}

		// modify columns
		if targetCol, ok := targetMapColumns[name]; ok && !utils.IsSameColumn(col, targetCol) {
			if col != targetCol {
				output := sourceSchema.CreateModifyColumn(sourceColumns, col)
				outputSql += output + "\n"

				fmt.Printf(" [✓] Modify column: %s in table: %s\n", name, tableName)
			}
		}
	}

	// drop columns
	for name, col := range targetMapColumns {
		if _, ok := sourceMapColumns[name]; !ok {
			output := sourceSchema.CreateDropColumn(tableName, col.ColumnName.String)
			outputSql += output + "\n"

			fmt.Printf(" [✓] Drop column: %s from table: %s\n", name, tableName)
		}
	}

	// if no changes, clear the outputSql for this table
	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql
}

func compareIndexes(sourceSchema, targetSchema *schema.Schema, tableName string) string {
	outputSql := ""

	sourceIndexes := sourceSchema.GetIndexes(tableName)
	targetIndexes := targetSchema.GetIndexes(tableName)

	targetMap := make(map[string]models.IndexInfo)
	for _, idx := range targetIndexes {
		key := idx.IndexName + "|" + idx.Columns
		targetMap[key] = idx
	}

	for _, idx := range sourceIndexes {
		key := idx.IndexName + "|" + idx.Columns
		if _, ok := targetMap[key]; !ok {
			output := sourceSchema.CreateAddIndex(idx)
			outputSql += output + "\n"

			fmt.Printf(" [✓] Create index: %s on table: %s\n", idx.IndexName, tableName)
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql
}

func compareForeignKeys(sourceSchema, targetSchema *schema.Schema, tableName string) string {
	outputSql := ""

	sourceFK := sourceSchema.GetForeignKeys(tableName)
	targetFK := targetSchema.GetForeignKeys(tableName)

	targetFKMap := make(map[string]models.ForeignKey)
	for _, fk := range targetFK {
		key := fk.ConstraintName + "|" + fk.ColumnName + "|" + fk.ReferencedTable
		targetFKMap[key] = fk
	}

	for _, fk := range sourceFK {
		key := fk.ConstraintName + "|" + fk.ColumnName + "|" + fk.ReferencedTable
		if _, ok := targetFKMap[key]; !ok {
			output := sourceSchema.CreateForeignKey(fk)
			outputSql += output + "\n"

			fmt.Printf(" [✓] Add foreign key: %s on table: %s\n", fk.ConstraintName, tableName)
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql
}

func outputSQL(sourceCfg, targetCfg models.DataSource) (outputSql string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	outputSql += "-- Generate from Synchroma at " + timestamp + "\n"
	outputSql += "-- Please review the SQL before applying it to the target database.\n"
	outputSql += "-- Source: \n"
	outputSql += "--   Host: " + sourceCfg.Host + "\n"
	outputSql += "--   Database: " + sourceCfg.DBName + "\n"
	outputSql += "-- Target: \n"
	outputSql += "--   Host: " + targetCfg.Host + "\n"
	outputSql += "--   Database: " + targetCfg.DBName + "\n"
	outputSql += "\n\n"

	return
}
