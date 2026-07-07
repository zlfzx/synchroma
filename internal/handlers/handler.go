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

	dropTables, _ := cmd.Flags().GetBool("drop-tables")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	apply, _ := cmd.Flags().GetBool("apply")

	// init config
	sourceCfg, targetCfg := config.GetConfig(cmd)

	var wg sync.WaitGroup
	s := chin.New().WithWait(&wg)
	go s.Start()

	// init source schema
	sourceSchema, err := schema.InitSchema(sourceCfg)
	if err != nil {
		s.Stop()
		wg.Wait()
		log.Fatalf("Failed to init source schema: %v", err)
	}
	defer sourceSchema.Close()

	fmt.Println("Connected to source database:")
	fmt.Println(" - Host:", sourceCfg.Host)
	fmt.Println(" - Database:", sourceCfg.DBName)
	fmt.Println(" ")

	// init target schema
	targetSchema, err := schema.InitSchema(targetCfg)
	if err != nil {
		s.Stop()
		wg.Wait()
		log.Fatalf("Failed to init target schema: %v", err)
	}
	defer targetSchema.Close()

	fmt.Println("Connected to target database:")
	fmt.Println(" - Host:", targetCfg.Host)
	fmt.Println(" - Database:", targetCfg.DBName)
	fmt.Println(" ")

	outputSql := outputSQL(sourceCfg, targetCfg)
	// disable foreign key checks
	outputSql += "SET FOREIGN_KEY_CHECKS=0;\n\n"

	// check tables
	tableDependencies, err := sourceSchema.GetTableDependencies()
	if err != nil {
		s.Stop()
		wg.Wait()
		log.Fatalf("Failed to get table dependencies: %v", err)
	}

	sourceTables, err := sourceSchema.GetTables()
	if err != nil {
		s.Stop()
		wg.Wait()
		log.Fatalf("Failed to get source tables: %v", err)
	}

	tables := []string{}
	for _, t := range sourceTables {
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

	targetTablesList, err := targetSchema.GetTables()
	if err != nil {
		s.Stop()
		wg.Wait()
		log.Fatalf("Failed to get target tables: %v", err)
	}

	targetTables := make(map[string]models.Table)
	for _, targetTable := range targetTablesList {
		targetTables[targetTable.TableName.String] = targetTable
	}

	for _, tableName := range orderedTables {
		if _, exists := targetTables[tableName]; !exists {
			// create table if not exists
			outputSql += "\n-- Table: " + tableName + "\n"
			output, err := sourceSchema.CreateTable(tableName)
			if err != nil {
				log.Printf("Failed to generate create table for %s: %v", tableName, err)
				continue
			}
			outputSql += output + "\n\n"

			fmt.Println(" [✓] Create table:", tableName)
			
			// Remove from target map so we can detect dropped tables later
			delete(targetTables, tableName)
		} else {
			// compare columns
			colSql, err := compareColumns(sourceSchema, targetSchema, tableName)
			if err != nil {
				log.Printf("Failed to compare columns for %s: %v", tableName, err)
			} else {
				outputSql += colSql
			}

			// compare indexes
			idxSql, err := compareIndexes(sourceSchema, targetSchema, tableName)
			if err != nil {
				log.Printf("Failed to compare indexes for %s: %v", tableName, err)
			} else {
				outputSql += idxSql
			}

			// compare foreign keys
			fkSql, err := compareForeignKeys(sourceSchema, targetSchema, tableName)
			if err != nil {
				log.Printf("Failed to compare foreign keys for %s: %v", tableName, err)
			} else {
				outputSql += fkSql
			}
			
			// Remove from target map
			delete(targetTables, tableName)
		}
	}

	// Drop tables that exist in target but not in source
	if dropTables {
		for tableName := range targetTables {
			dropSql := targetSchema.CreateDropTable(tableName)
			outputSql += "\n-- Table: " + tableName + "\n" + dropSql + "\n"
			fmt.Printf(" [✓] Drop table: %s\n", tableName)
		}
	}

	// drop foreign key checks at the end of the sql file
	outputSql += "\nSET FOREIGN_KEY_CHECKS=1;\n"

	s.Stop()
	wg.Wait()

	if dryRun {
		fmt.Println("\n--- DRY RUN OUTPUT ---")
		fmt.Println(outputSql)
		return
	}

	if apply {
		fmt.Println("\nApplying SQL to target database...")
		// We would need a method to execute raw SQL on the provider.
		// Since we didn't add it to SchemaProvider yet, we can access targetSchema (*schema.MySQLSchema) directly for now
		// or add it to the interface. Let's add an ExecuteSQL method to SchemaProvider or do it manually if possible.
		// For simplicity, we just print a message that it's applied (or we can add ExecuteSQL to interface).
		if mysqlSchema, ok := targetSchema.(*schema.MySQLSchema); ok {
			_, err := mysqlSchema.DB.Exec(outputSql)
			if err != nil {
				log.Fatalf("Failed to apply SQL: %v", err)
			}
			fmt.Println("SQL successfully applied to target database!")
		} else {
			log.Fatalf("Apply mode is currently only supported for MySQL target")
		}
		return
	}

	// filename
	filename, _ := cmd.Flags().GetString("output-file")
	if filename == "" {
		filename = sourceCfg.DBName + "_to_" + targetCfg.DBName + ".sql"
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	pathFile := wd + "/" + filename

	f, err := os.Create(pathFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(outputSql)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	fmt.Println(" ")
	fmt.Println("Success!\nSQL file has been generated at " + pathFile)
}

func compareColumns(sourceSchema, targetSchema schema.SchemaProvider, tableName string) (string, error) {
	outputSql := ""

	sourceColumns, err := sourceSchema.GetColumns(tableName)
	if err != nil {
		return "", err
	}
	
	sourceMapColumns := make(map[string]models.Column)
	for _, sourceColumn := range sourceColumns {
		sourceMapColumns[sourceColumn.ColumnName.String] = sourceColumn
	}

	targetColumns, err := targetSchema.GetColumns(tableName)
	if err != nil {
		return "", err
	}
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
			output := sourceSchema.CreateModifyColumn(sourceColumns, col)
			outputSql += output + "\n"

			fmt.Printf(" [✓] Modify column: %s in table: %s\n", name, tableName)
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

	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql, nil
}

func compareIndexes(sourceSchema, targetSchema schema.SchemaProvider, tableName string) (string, error) {
	outputSql := ""

	sourceIndexes, err := sourceSchema.GetIndexes(tableName)
	if err != nil {
		return "", err
	}
	targetIndexes, err := targetSchema.GetIndexes(tableName)
	if err != nil {
		return "", err
	}

	sourceMap := make(map[string]models.IndexInfo)
	for _, idx := range sourceIndexes {
		key := idx.IndexName + "|" + idx.Columns
		sourceMap[key] = idx
	}

	targetMap := make(map[string]models.IndexInfo)
	for _, idx := range targetIndexes {
		key := idx.IndexName + "|" + idx.Columns
		targetMap[key] = idx
	}

	// Add missing indexes
	for key, idx := range sourceMap {
		if _, ok := targetMap[key]; !ok {
			output := sourceSchema.CreateAddIndex(idx)
			outputSql += output + "\n"
			fmt.Printf(" [✓] Create index: %s on table: %s\n", idx.IndexName, tableName)
		}
	}
	
	// Drop indexes that exist in target but not in source
	for key, idx := range targetMap {
		// Ignore PRIMARY key dropping as it usually drops with column drops
		if idx.IndexName == "PRIMARY" {
			continue
		}
		if _, ok := sourceMap[key]; !ok {
			output := sourceSchema.CreateDropIndex(tableName, idx.IndexName)
			outputSql += output + "\n"
			fmt.Printf(" [✓] Drop index: %s on table: %s\n", idx.IndexName, tableName)
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql, nil
}

func compareForeignKeys(sourceSchema, targetSchema schema.SchemaProvider, tableName string) (string, error) {
	outputSql := ""

	sourceFK, err := sourceSchema.GetForeignKeys(tableName)
	if err != nil {
		return "", err
	}
	targetFK, err := targetSchema.GetForeignKeys(tableName)
	if err != nil {
		return "", err
	}

	sourceFKMap := make(map[string]models.ForeignKey)
	for _, fk := range sourceFK {
		key := fk.ConstraintName + "|" + fk.ColumnName + "|" + fk.ReferencedTable
		sourceFKMap[key] = fk
	}

	targetFKMap := make(map[string]models.ForeignKey)
	for _, fk := range targetFK {
		key := fk.ConstraintName + "|" + fk.ColumnName + "|" + fk.ReferencedTable
		targetFKMap[key] = fk
	}

	// Add missing foreign keys
	for key, fk := range sourceFKMap {
		if _, ok := targetFKMap[key]; !ok {
			output := sourceSchema.CreateForeignKey(fk)
			outputSql += output + "\n"
			fmt.Printf(" [✓] Add foreign key: %s on table: %s\n", fk.ConstraintName, tableName)
		}
	}
	
	// Drop foreign keys that exist in target but not in source
	for key, fk := range targetFKMap {
		if _, ok := sourceFKMap[key]; !ok {
			output := sourceSchema.CreateDropForeignKey(tableName, fk.ConstraintName)
			outputSql += output + "\n"
			fmt.Printf(" [✓] Drop foreign key: %s on table: %s\n", fk.ConstraintName, tableName)
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table: " + tableName + "\n" + outputSql + "\n"
	}

	return outputSql, nil
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
