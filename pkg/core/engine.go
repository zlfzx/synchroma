package core

import (
	"fmt"
	"strings"
	"sync"
	"synchroma/pkg/models"
	"synchroma/pkg/schema"
	"synchroma/pkg/utils"
	"time"
)

type SyncOptions struct {
	SourceCfg  models.DataSource
	TargetCfg  models.DataSource
	DropTables bool
	OnProgress func(msg string)
}

type SyncStats struct {
	mu               sync.Mutex
	TablesAdded      int
	TablesModified   int
	TablesDropped    int
	ColumnsAdded     int
	ColumnsModified  int
	ColumnsDropped   int
	IndexesAdded     int
	IndexesDropped   int
	FKsAdded         int
	FKsDropped       int
	TablePropsSynced int
	ViewsSynced      int
	TriggersSynced   int
	RoutinesSynced   int
}

type SyncResult struct {
	SQL   string
	Stats *SyncStats
}

func GenerateSyncSQL(opts SyncOptions) (*SyncResult, error) {
	logMsg := func(msg string) {
		if opts.OnProgress != nil {
			opts.OnProgress(msg)
		}
	}

	sourceSchema, err := schema.InitSchema(opts.SourceCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init source schema: %w", err)
	}
	defer sourceSchema.Close()

	targetSchema, err := schema.InitSchema(opts.TargetCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init target schema: %w", err)
	}
	defer targetSchema.Close()

	outputSql := outputSQL(opts.SourceCfg, opts.TargetCfg)
	outputSql += "SET FOREIGN_KEY_CHECKS=0;\n\n"

	var stats SyncStats

	// 1. Prepare Tables
	tableDependencies, err := sourceSchema.GetTableDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to get table dependencies: %w", err)
	}

	sourceTablesList, err := sourceSchema.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get source tables: %w", err)
	}

	sourceTablesMap := make(map[string]models.Table)
	tables := []string{}
	for _, t := range sourceTablesList {
		tables = append(tables, t.TableName.String)
		sourceTablesMap[t.TableName.String] = t
	}

	tableIndex := make(map[string]int)
	for i, table := range tables {
		tableIndex[table] = i
	}

	graph := utils.BuildDependencyGraph(tables, tableDependencies)
	orderedTables := utils.TopologicalSort(graph, tableIndex)

	targetTablesList, err := targetSchema.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get target tables: %w", err)
	}

	targetTablesMap := make(map[string]models.Table)
	for _, targetTable := range targetTablesList {
		targetTablesMap[targetTable.TableName.String] = targetTable
	}

	logMsg("Processing tables in parallel...")

	// 2. Parallel Processing
	tableOutputs := make([]string, len(orderedTables))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit to 10 concurrent routines
	var printMu sync.Mutex

	for i, tableName := range orderedTables {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var localOutput string

			if _, exists := targetTablesMap[name]; !exists {
				localOutput += "\n-- Table: " + name + "\n"
				output, err := sourceSchema.CreateTable(name)
				if err != nil {
					logMsg(fmt.Sprintf("Failed to generate create table for %s: %v", name, err))
					return
				}
				localOutput += output + "\n\n"

				printMu.Lock()
				logMsg(fmt.Sprintf(" [✓] Create table: %s", name))
				printMu.Unlock()

				stats.mu.Lock()
				stats.TablesAdded++
				stats.mu.Unlock()
			} else {
				tableModified := false

				// Properties
				srcTbl := sourceTablesMap[name]
				tgtTbl := targetTablesMap[name]
				var props []string
				if srcTbl.Engine.String != tgtTbl.Engine.String && srcTbl.Engine.Valid {
					props = append(props, "ENGINE="+srcTbl.Engine.String)
				}
				if srcTbl.TableCollation.String != tgtTbl.TableCollation.String && srcTbl.TableCollation.Valid {
					props = append(props, "COLLATE="+srcTbl.TableCollation.String)
				}
				if srcTbl.TableComment.String != tgtTbl.TableComment.String {
					escapedComment := strings.ReplaceAll(srcTbl.TableComment.String, "'", "''")
					props = append(props, fmt.Sprintf("COMMENT='%s'", escapedComment))
				}
				if len(props) > 0 {
					propSql := sourceSchema.CreateAlterTableProperties(name, props)
					localOutput += "\n-- Table Properties: " + name + "\n" + propSql + "\n"

					printMu.Lock()
					logMsg(fmt.Sprintf(" [✓] Update properties for table: %s", name))
					printMu.Unlock()

					stats.mu.Lock()
					stats.TablePropsSynced++
					tableModified = true
					stats.mu.Unlock()
				}

				// Columns
				colSql, err := compareColumns(sourceSchema, targetSchema, name, &stats, &printMu, logMsg)
				if err != nil {
					logMsg(fmt.Sprintf("Failed to compare columns for %s: %v", name, err))
				} else if colSql != "" {
					localOutput += colSql
					tableModified = true
				}

				// Indexes
				idxSql, err := compareIndexes(sourceSchema, targetSchema, name, &stats, &printMu, logMsg)
				if err != nil {
					logMsg(fmt.Sprintf("Failed to compare indexes for %s: %v", name, err))
				} else if idxSql != "" {
					localOutput += idxSql
					tableModified = true
				}

				// FKs
				fkSql, err := compareForeignKeys(sourceSchema, targetSchema, name, &stats, &printMu, logMsg)
				if err != nil {
					logMsg(fmt.Sprintf("Failed to compare foreign keys for %s: %v", name, err))
				} else if fkSql != "" {
					localOutput += fkSql
					tableModified = true
				}

				if tableModified {
					stats.mu.Lock()
					stats.TablesModified++
					stats.mu.Unlock()
				}
			}

			tableOutputs[index] = localOutput
		}(i, tableName)
	}

	wg.Wait()

	for _, sqlBlock := range tableOutputs {
		outputSql += sqlBlock
	}

	// 3. Drop Tables
	if opts.DropTables {
		for targetName := range targetTablesMap {
			if _, exists := sourceTablesMap[targetName]; !exists {
				dropSql := targetSchema.CreateDropTable(targetName)
				outputSql += "\n-- Drop Table: " + targetName + "\n" + dropSql + "\n"
				logMsg(fmt.Sprintf(" [✓] Drop table: %s", targetName))

				stats.mu.Lock()
				stats.TablesDropped++
				stats.mu.Unlock()
			}
		}
	}

	// 4. Advanced Objects
	logMsg("Processing advanced objects (Views, Triggers, Routines)...")
	viewsSql, err := compareAdvancedObjects(sourceSchema, targetSchema, "VIEW", &stats, &printMu, logMsg)
	if err == nil && viewsSql != "" {
		outputSql += viewsSql
	}

	triggersSql, err := compareAdvancedObjects(sourceSchema, targetSchema, "TRIGGER", &stats, &printMu, logMsg)
	if err == nil && triggersSql != "" {
		outputSql += triggersSql
	}

	routinesSql, err := compareAdvancedObjects(sourceSchema, targetSchema, "ROUTINE", &stats, &printMu, logMsg)
	if err == nil && routinesSql != "" {
		outputSql += routinesSql
	}

	outputSql += "\nSET FOREIGN_KEY_CHECKS=1;\n"

	return &SyncResult{
		SQL:   outputSql,
		Stats: &stats,
	}, nil
}

func compareColumns(sourceSchema, targetSchema schema.SchemaProvider, tableName string, stats *SyncStats, printMu *sync.Mutex, logMsg func(string)) (string, error) {
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

	for name, col := range sourceMapColumns {
		if _, ok := targetMapColumns[name]; !ok {
			outputSql += sourceSchema.CreateAddColumn(sourceColumns, col) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Add column: %s to table: %s", name, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.ColumnsAdded++
			stats.mu.Unlock()
		} else if targetCol, ok := targetMapColumns[name]; ok && !utils.IsSameColumn(col, targetCol) {
			outputSql += sourceSchema.CreateModifyColumn(sourceColumns, col) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Modify column: %s in table: %s", name, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.ColumnsModified++
			stats.mu.Unlock()
		}
	}

	for name, col := range targetMapColumns {
		if _, ok := sourceMapColumns[name]; !ok {
			outputSql += sourceSchema.CreateDropColumn(tableName, col.ColumnName.String) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Drop column: %s from table: %s", name, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.ColumnsDropped++
			stats.mu.Unlock()
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table Columns: " + tableName + "\n" + outputSql + "\n"
	}
	return outputSql, nil
}

func compareIndexes(sourceSchema, targetSchema schema.SchemaProvider, tableName string, stats *SyncStats, printMu *sync.Mutex, logMsg func(string)) (string, error) {
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
		sourceMap[idx.IndexName+"|"+idx.Columns] = idx
	}
	targetMap := make(map[string]models.IndexInfo)
	for _, idx := range targetIndexes {
		targetMap[idx.IndexName+"|"+idx.Columns] = idx
	}

	for key, idx := range sourceMap {
		if _, ok := targetMap[key]; !ok {
			outputSql += sourceSchema.CreateAddIndex(idx) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Create index: %s on table: %s", idx.IndexName, tableName) )
			printMu.Unlock()

			stats.mu.Lock()
			stats.IndexesAdded++
			stats.mu.Unlock()
		}
	}

	for key, idx := range targetMap {
		if idx.IndexName == "PRIMARY" {
			continue
		}
		if _, ok := sourceMap[key]; !ok {
			outputSql += sourceSchema.CreateDropIndex(tableName, idx.IndexName) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Drop index: %s on table: %s", idx.IndexName, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.IndexesDropped++
			stats.mu.Unlock()
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table Indexes: " + tableName + "\n" + outputSql + "\n"
	}
	return outputSql, nil
}

func compareForeignKeys(sourceSchema, targetSchema schema.SchemaProvider, tableName string, stats *SyncStats, printMu *sync.Mutex, logMsg func(string)) (string, error) {
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
		sourceFKMap[fk.ConstraintName+"|"+fk.ColumnName+"|"+fk.ReferencedTable] = fk
	}
	targetFKMap := make(map[string]models.ForeignKey)
	for _, fk := range targetFK {
		targetFKMap[fk.ConstraintName+"|"+fk.ColumnName+"|"+fk.ReferencedTable] = fk
	}

	for key, fk := range sourceFKMap {
		if _, ok := targetFKMap[key]; !ok {
			outputSql += sourceSchema.CreateForeignKey(fk) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Add foreign key: %s on table: %s", fk.ConstraintName, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.FKsAdded++
			stats.mu.Unlock()
		}
	}

	for key, fk := range targetFKMap {
		if _, ok := sourceFKMap[key]; !ok {
			outputSql += sourceSchema.CreateDropForeignKey(tableName, fk.ConstraintName) + "\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Drop foreign key: %s on table: %s", fk.ConstraintName, tableName))
			printMu.Unlock()

			stats.mu.Lock()
			stats.FKsDropped++
			stats.mu.Unlock()
		}
	}

	if outputSql != "" {
		outputSql = "\n-- Table FKs: " + tableName + "\n" + outputSql + "\n"
	}
	return outputSql, nil
}

func compareAdvancedObjects(sourceSchema, targetSchema schema.SchemaProvider, objectType string, stats *SyncStats, printMu *sync.Mutex, logMsg func(string)) (string, error) {
	outputSql := ""

	var sourceObjs, targetObjs []models.SchemaObject
	var err error

	switch objectType {
	case "VIEW":
		sourceObjs, err = sourceSchema.GetViews()
		if err == nil {
			targetObjs, err = targetSchema.GetViews()
		}
	case "TRIGGER":
		sourceObjs, err = sourceSchema.GetTriggers()
		if err == nil {
			targetObjs, err = targetSchema.GetTriggers()
		}
	case "ROUTINE":
		sourceObjs, err = sourceSchema.GetRoutines()
		if err == nil {
			targetObjs, err = targetSchema.GetRoutines()
		}
	}

	if err != nil {
		logMsg(fmt.Sprintf("Failed to fetch %ss: %v", objectType, err))
		return "", err
	}

	sourceMap := make(map[string]models.SchemaObject)
	for _, obj := range sourceObjs {
		sourceMap[obj.Name] = obj
	}
	targetMap := make(map[string]models.SchemaObject)
	for _, obj := range targetObjs {
		targetMap[obj.Name] = obj
	}

	for name, obj := range sourceMap {
		var srcDef, tgtDef string
		var getErr error
		switch objectType {
		case "VIEW":
			srcDef, getErr = sourceSchema.GetViewDefinition(name)
		case "TRIGGER":
			srcDef, getErr = sourceSchema.GetTriggerDefinition(name)
		case "ROUTINE":
			srcDef, getErr = sourceSchema.GetRoutineDefinition(name, obj.Type)
		}
		if getErr != nil {
			logMsg(fmt.Sprintf("Failed to get %s definition for %s: %v", objectType, name, getErr))
			continue
		}

		if _, ok := targetMap[name]; ok {
			switch objectType {
			case "VIEW":
				tgtDef, _ = targetSchema.GetViewDefinition(name)
			case "TRIGGER":
				tgtDef, _ = targetSchema.GetTriggerDefinition(name)
			case "ROUTINE":
				tgtDef, _ = targetSchema.GetRoutineDefinition(name, obj.Type)
			}
		}

		if tgtDef != srcDef {
			if tgtDef != "" {
				switch objectType {
				case "VIEW":
					outputSql += sourceSchema.CreateDropView(name) + "\n"
				case "TRIGGER":
					outputSql += sourceSchema.CreateDropTrigger(name) + "\n"
				case "ROUTINE":
					outputSql += sourceSchema.CreateDropRoutine(name, obj.Type) + "\n"
				}
			}

			outputSql += srcDef + "\n\n"

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Sync %s: %s", objectType, name))
			printMu.Unlock()

			stats.mu.Lock()
			switch objectType {
			case "VIEW":
				stats.ViewsSynced++
			case "TRIGGER":
				stats.TriggersSynced++
			case "ROUTINE":
				stats.RoutinesSynced++
			}
			stats.mu.Unlock()
		}
	}

	for name, obj := range targetMap {
		if _, ok := sourceMap[name]; !ok {
			switch objectType {
			case "VIEW":
				outputSql += sourceSchema.CreateDropView(name) + "\n"
			case "TRIGGER":
				outputSql += sourceSchema.CreateDropTrigger(name) + "\n"
			case "ROUTINE":
				outputSql += sourceSchema.CreateDropRoutine(name, obj.Type) + "\n"
			}

			printMu.Lock()
			logMsg(fmt.Sprintf(" [✓] Drop %s: %s", objectType, name))
			printMu.Unlock()
		}
	}

	if outputSql != "" {
		outputSql = "\n-- " + objectType + "S --\n" + outputSql
	}
	return outputSql, nil
}

func outputSQL(sourceCfg, targetCfg models.DataSource) (outputSql string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	outputSql += "-- Generate from Synchroma at " + timestamp + "\n"
	outputSql += "-- Source: Host=" + sourceCfg.Host + " | DB=" + sourceCfg.DBName + "\n"
	outputSql += "-- Target: Host=" + targetCfg.Host + " | DB=" + targetCfg.DBName + "\n\n"
	return
}
