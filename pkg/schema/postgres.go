package schema

import (
	"fmt"
	"strings"
	"synchroma/pkg/models"
	"synchroma/pkg/utils"

	"github.com/jmoiron/sqlx"
)

type PostgresSchema struct {
	DB     *sqlx.DB
	DBName string
	Tables []models.Table
}

func (s *PostgresSchema) GetTableDependencies() (map[string][]string, error) {
	sql := `
		SELECT
			tc.table_name AS "TABLE_NAME",
			ccu.table_name AS "REFERENCED_TABLE_NAME"
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.constraint_schema = ccu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
		ORDER BY tc.table_name, ccu.table_name
	`

	var dependencies []models.TableDependency
	err := s.DB.Select(&dependencies, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get table dependencies: %w", err)
	}

	tableDependencies := make(map[string][]string)
	for _, dep := range dependencies {
		if _, ok := tableDependencies[dep.TableName.String]; !ok {
			tableDependencies[dep.TableName.String] = []string{}
		}
		tableDependencies[dep.TableName.String] = append(tableDependencies[dep.TableName.String], dep.ReferencedTable.String)

		if _, ok := tableDependencies[dep.ReferencedTable.String]; !ok {
			tableDependencies[dep.ReferencedTable.String] = []string{}
		}
	}

	tables, err := s.GetTables()
	if err != nil {
		return nil, err
	}

	for _, t := range tables {
		if _, ok := tableDependencies[t.TableName.String]; !ok {
			tableDependencies[t.TableName.String] = []string{}
		}
	}

	return tableDependencies, nil
}

func (s *PostgresSchema) GetTables() ([]models.Table, error) {
	if s.Tables != nil {
		return s.Tables, nil
	}

	sql := `
		SELECT 
			t.table_catalog AS "TABLE_CATALOG",
			t.table_schema AS "TABLE_SCHEMA",
			t.table_name AS "TABLE_NAME",
			t.table_type AS "TABLE_TYPE",
			COALESCE(obj_description((t.table_schema || '.' || t.table_name)::regclass), '') AS "TABLE_COMMENT"
		FROM information_schema.tables t
		WHERE t.table_schema = 'public'
			AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name
	`
	var tables []models.Table
	err := s.DB.Select(&tables, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	s.Tables = tables
	return tables, nil
}

func (s *PostgresSchema) CreateTable(tableName string) (string, error) {
	columns, err := s.GetColumns(tableName)
	if err != nil {
		return "", err
	}

	var colDefs []string
	for _, col := range columns {
		def := fmt.Sprintf("    %s %s", utils.EscapeIdentifierPG(col.ColumnName.String), col.ColumnType)

		if col.ColumnDefault.Valid && col.ColumnDefault.String != "" {
			def += " DEFAULT " + col.ColumnDefault.String
		}

		if col.IsNullable == "NO" {
			def += " NOT NULL"
		}

		colDefs = append(colDefs, def)
	}

	// Primary key
	pkSQL := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
		ORDER BY kcu.ordinal_position
	`
	var pkCols []string
	if err := s.DB.Select(&pkCols, pkSQL, tableName); err == nil && len(pkCols) > 0 {
		escaped := make([]string, len(pkCols))
		for i, c := range pkCols {
			escaped[i] = utils.EscapeIdentifierPG(c)
		}
		colDefs = append(colDefs, fmt.Sprintf("    PRIMARY KEY (%s)", strings.Join(escaped, ", ")))
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (\n%s\n);",
		utils.EscapeIdentifierPG(tableName),
		strings.Join(colDefs, ",\n"),
	)

	return createSQL, nil
}

func (s *PostgresSchema) GetColumns(tableName string) ([]models.Column, error) {
	sql := `
		SELECT 
			c.table_catalog AS "TABLE_CATALOG",
			c.table_schema AS "TABLE_SCHEMA",
			c.table_name AS "TABLE_NAME",
			c.column_name AS "COLUMN_NAME",
			c.ordinal_position AS "ORDINAL_POSITION",
			c.column_default AS "COLUMN_DEFAULT",
			c.is_nullable AS "IS_NULLABLE",
			c.data_type AS "DATA_TYPE",
			c.character_maximum_length AS "CHARACTER_MAXIMUM_LENGTH",
			c.character_octet_length AS "CHARACTER_OCTET_LENGTH",
			c.numeric_precision AS "NUMERIC_PRECISION",
			c.numeric_scale AS "NUMERIC_SCALE",
			c.datetime_precision AS "DATETIME_PRECISION",
			c.character_set_name AS "CHARACTER_SET_NAME",
			c.collation_name AS "COLLATION_NAME",
			(CASE 
				WHEN c.data_type = 'character varying' THEN 'varchar(' || c.character_maximum_length || ')'
				WHEN c.data_type = 'character' THEN 'char(' || c.character_maximum_length || ')'
				WHEN c.data_type = 'numeric' THEN 'numeric(' || c.numeric_precision || ',' || c.numeric_scale || ')'
				ELSE c.data_type
			END) AS "COLUMN_TYPE",
			'' AS "COLUMN_KEY",
			COALESCE(
				CASE WHEN c.column_default LIKE 'nextval%%' THEN 'auto_increment' ELSE '' END,
				''
			) AS "EXTRA",
			'' AS "PRIVILEGES",
			COALESCE(pgd.description, '') AS "COLUMN_COMMENT",
			COALESCE(c.generation_expression, '') AS "GENERATION_EXPRESSION"
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st
			ON c.table_schema = st.schemaname AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pgd
			ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = 'public' AND c.table_name = $1
		ORDER BY c.ordinal_position
	`
	var columns []models.Column
	if err := s.DB.Select(&columns, sql, tableName); err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}

	return columns, nil
}

func (s *PostgresSchema) buildColumnSQL(action string, columns []models.Column, col models.Column) string {
	colType := col.ColumnType

	nullable := "NOT NULL"
	if col.IsNullable == "YES" {
		nullable = ""
	}

	defaultValue := ""
	if col.ColumnDefault.Valid && col.ColumnDefault.String != "" {
		defaultValue = "DEFAULT " + col.ColumnDefault.String
	}

	if action == "ADD" {
		sql := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s %s %s;",
			utils.EscapeIdentifierPG(col.TableName.String),
			utils.EscapeIdentifierPG(col.ColumnName.String),
			colType,
			nullable,
			defaultValue,
		)
		return sql
	}

	// PostgreSQL doesn't have MODIFY COLUMN, we need multiple ALTER statements
	var stmts []string

	// Change type
	stmts = append(stmts, fmt.Sprintf(
		"ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
		utils.EscapeIdentifierPG(col.TableName.String),
		utils.EscapeIdentifierPG(col.ColumnName.String),
		colType,
	))

	// Change nullable
	if col.IsNullable == "YES" {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
			utils.EscapeIdentifierPG(col.TableName.String),
			utils.EscapeIdentifierPG(col.ColumnName.String),
		))
	} else {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
			utils.EscapeIdentifierPG(col.TableName.String),
			utils.EscapeIdentifierPG(col.ColumnName.String),
		))
	}

	// Change default
	if col.ColumnDefault.Valid && col.ColumnDefault.String != "" {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
			utils.EscapeIdentifierPG(col.TableName.String),
			utils.EscapeIdentifierPG(col.ColumnName.String),
			col.ColumnDefault.String,
		))
	} else {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
			utils.EscapeIdentifierPG(col.TableName.String),
			utils.EscapeIdentifierPG(col.ColumnName.String),
		))
	}

	return strings.Join(stmts, "\n")
}

func (s *PostgresSchema) CreateAddColumn(columns []models.Column, col models.Column) string {
	return s.buildColumnSQL("ADD", columns, col)
}

func (s *PostgresSchema) CreateModifyColumn(columns []models.Column, col models.Column) string {
	return s.buildColumnSQL("MODIFY", columns, col)
}

func (s *PostgresSchema) CreateDropColumn(tableName, columnName string) string {
	return fmt.Sprintf(
		"ALTER TABLE %s DROP COLUMN %s;",
		utils.EscapeIdentifierPG(tableName),
		utils.EscapeIdentifierPG(columnName),
	)
}

func (s *PostgresSchema) GetIndexes(tableName string) ([]models.IndexInfo, error) {
	sql := `
		SELECT 
			t.relname AS "TABLE_NAME",
			i.relname AS "INDEX_NAME",
			string_agg(a.attname, ',' ORDER BY array_position(ix.indkey, a.attnum)) AS "COLUMNS",
			CASE WHEN ix.indisunique THEN 0 ELSE 1 END AS "NON_UNIQUE"
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = 'public'
			AND t.relname = $1
			AND NOT ix.indisprimary
		GROUP BY t.relname, i.relname, ix.indisunique
	`

	var indexes []models.IndexInfo
	err := s.DB.Select(&indexes, sql, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
	}

	return indexes, nil
}

func (s *PostgresSchema) CreateAddIndex(index models.IndexInfo) string {
	indexType := "INDEX"
	if index.NonUnique == 0 {
		indexType = "UNIQUE INDEX"
	}

	cols := strings.Split(index.Columns, ",")
	for i, c := range cols {
		cols[i] = utils.EscapeIdentifierPG(strings.TrimSpace(c))
	}
	escapedCols := strings.Join(cols, ", ")

	return fmt.Sprintf(
		"CREATE %s %s ON %s (%s);",
		indexType,
		utils.EscapeIdentifierPG(index.IndexName),
		utils.EscapeIdentifierPG(index.TableName),
		escapedCols,
	)
}

func (s *PostgresSchema) CreateDropIndex(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s;", utils.EscapeIdentifierPG(indexName))
}

func (s *PostgresSchema) CreateAlterTableProperties(tableName string, props []string) string {
	var stmts []string
	for _, prop := range props {
		if strings.HasPrefix(prop, "COMMENT=") {
			comment := strings.TrimPrefix(prop, "COMMENT=")
			comment = strings.Trim(comment, "'")
			stmts = append(stmts, fmt.Sprintf("COMMENT ON TABLE %s IS '%s';",
				utils.EscapeIdentifierPG(tableName), comment))
		}
	}
	return strings.Join(stmts, "\n")
}

func (s *PostgresSchema) GetForeignKeys(tableName string) ([]models.ForeignKey, error) {
	sql := `
		SELECT
			tc.constraint_name AS "CONSTRAINT_NAME",
			tc.table_name AS "TABLE_NAME",
			kcu.column_name AS "COLUMN_NAME",
			ccu.table_name AS "REFERENCED_TABLE_NAME",
			ccu.column_name AS "REFERENCED_COLUMN_NAME",
			rc.update_rule AS "UPDATE_RULE",
			rc.delete_rule AS "DELETE_RULE"
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`

	var foreignKeys []models.ForeignKey
	err := s.DB.Select(&foreignKeys, sql, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", tableName, err)
	}

	return foreignKeys, nil
}

func (s *PostgresSchema) CreateForeignKey(fk models.ForeignKey) string {
	return fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE %s ON DELETE %s;",
		utils.EscapeIdentifierPG(fk.TableName),
		utils.EscapeIdentifierPG(fk.ConstraintName),
		utils.EscapeIdentifierPG(fk.ColumnName),
		utils.EscapeIdentifierPG(fk.ReferencedTable),
		utils.EscapeIdentifierPG(fk.ReferencedColumn),
		fk.UpdateRule,
		fk.DeleteRule,
	)
}

func (s *PostgresSchema) CreateDropForeignKey(tableName, constraintName string) string {
	return fmt.Sprintf(
		"ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;",
		utils.EscapeIdentifierPG(tableName),
		utils.EscapeIdentifierPG(constraintName),
	)
}

func (s *PostgresSchema) CreateDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", utils.EscapeIdentifierPG(tableName))
}

func (s *PostgresSchema) GetViews() ([]models.SchemaObject, error) {
	sql := `
		SELECT viewname AS "NAME", 'VIEW' AS "TYPE"
		FROM pg_views
		WHERE schemaname = 'public'
	`
	var objs []models.SchemaObject
	if err := s.DB.Select(&objs, sql); err != nil {
		return nil, err
	}
	return objs, nil
}

func (s *PostgresSchema) GetTriggers() ([]models.SchemaObject, error) {
	sql := `
		SELECT DISTINCT trigger_name AS "NAME", 'TRIGGER' AS "TYPE"
		FROM information_schema.triggers
		WHERE trigger_schema = 'public'
	`
	var objs []models.SchemaObject
	if err := s.DB.Select(&objs, sql); err != nil {
		return nil, err
	}
	return objs, nil
}

func (s *PostgresSchema) GetRoutines() ([]models.SchemaObject, error) {
	sql := `
		SELECT routine_name AS "NAME",
			   CASE routine_type
				   WHEN 'FUNCTION' THEN 'FUNCTION'
				   WHEN 'PROCEDURE' THEN 'PROCEDURE'
				   ELSE 'FUNCTION'
			   END AS "TYPE"
		FROM information_schema.routines
		WHERE routine_schema = 'public'
			AND routine_name NOT LIKE 'pg_%'
	`
	var objs []models.SchemaObject
	if err := s.DB.Select(&objs, sql); err != nil {
		return nil, err
	}
	return objs, nil
}

func (s *PostgresSchema) GetViewDefinition(name string) (string, error) {
	sql := `SELECT definition FROM pg_views WHERE schemaname = 'public' AND viewname = $1`
	var definition string
	if err := s.DB.Get(&definition, sql, name); err != nil {
		return "", err
	}
	return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s", utils.EscapeIdentifierPG(name), strings.TrimSpace(definition)), nil
}

func (s *PostgresSchema) GetTriggerDefinition(name string) (string, error) {
	sql := `SELECT pg_get_triggerdef(t.oid) FROM pg_trigger t
			JOIN pg_class c ON t.tgrelid = c.oid
			JOIN pg_namespace n ON c.relnamespace = n.oid
			WHERE n.nspname = 'public' AND t.tgname = $1 AND NOT t.tgisinternal`
	var definition string
	if err := s.DB.Get(&definition, sql, name); err != nil {
		return "", err
	}
	return definition + ";", nil
}

func (s *PostgresSchema) GetRoutineDefinition(name, routineType string) (string, error) {
	sql := `SELECT pg_get_functiondef(p.oid) FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = 'public' AND p.proname = $1`
	var definition string
	if err := s.DB.Get(&definition, sql, name); err != nil {
		return "", err
	}
	return definition + ";", nil
}

func (s *PostgresSchema) CreateDropView(name string) string {
	return fmt.Sprintf("DROP VIEW IF EXISTS %s;", utils.EscapeIdentifierPG(name))
}

func (s *PostgresSchema) CreateDropTrigger(name string) string {
	// PostgreSQL requires the table name for DROP TRIGGER, use a safe pattern
	return fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", utils.EscapeIdentifierPG(name), utils.EscapeIdentifierPG(name))
}

func (s *PostgresSchema) CreateDropRoutine(name, routineType string) string {
	if routineType == "PROCEDURE" {
		return fmt.Sprintf("DROP PROCEDURE IF EXISTS %s;", utils.EscapeIdentifierPG(name))
	}
	return fmt.Sprintf("DROP FUNCTION IF EXISTS %s;", utils.EscapeIdentifierPG(name))
}

func (s *PostgresSchema) DisableFKChecks() string {
	return "SET session_replication_role = 'replica';"
}

func (s *PostgresSchema) EnableFKChecks() string {
	return "SET session_replication_role = 'origin';"
}

func (s *PostgresSchema) Close() error {
	return s.DB.Close()
}
