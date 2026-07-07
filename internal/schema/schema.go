package schema

import (
	"fmt"
	"regexp"
	"strings"
	"synchroma/internal/models"
	"synchroma/internal/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type MySQLSchema struct {
	DB     *sqlx.DB
	DBName string
	Tables []models.Table
}

func InitSchema(config models.DataSource) (SchemaProvider, error) {
	if config.Database == "" || config.Database != "mysql" {
		return nil, fmt.Errorf("database %s not supported", config.Database)
	}

	datasource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)

	DBSource, err := sqlx.Connect(config.Database, datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &MySQLSchema{
		DB:     DBSource,
		DBName: config.DBName,
	}, nil
}

func (s *MySQLSchema) GetTableDependencies() (map[string][]string, error) {
	sql := `
		SELECT 
			TABLE_NAME, REFERENCED_TABLE_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE 
			REFERENCED_TABLE_NAME IS NOT NULL
			AND TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, REFERENCED_TABLE_NAME
	`

	var dependencies []models.TableDependency
	err := s.DB.Select(&dependencies, sql, s.DBName)
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

	// ensure all tables are in the map
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

func (s *MySQLSchema) GetTables() ([]models.Table, error) {
	if s.Tables != nil {
		return s.Tables, nil
	}

	sql := "SELECT * FROM information_schema.tables WHERE table_schema = ?"
	var tables []models.Table
	err := s.DB.Select(&tables, sql, s.DBName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	s.Tables = tables
	return tables, nil
}

func (s *MySQLSchema) CreateTable(tableName string) (string, error) {
	var ddl models.CreateTable

	sql := "SHOW CREATE TABLE " + utils.EscapeIdentifier(tableName)
	if err := s.DB.Get(&ddl, sql); err != nil {
		return "", fmt.Errorf("failed to show create table %s: %w", tableName, err)
	}

	createTable := ddl.CreateTable

	// remove auto increment
	re := regexp.MustCompile(` AUTO_INCREMENT=\d+`)
	createTable = re.ReplaceAllString(createTable, "")

	createTable += ";"

	return createTable, nil
}

func (s *MySQLSchema) GetColumns(tableName string) ([]models.Column, error) {
	sql := "SELECT * FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ORDINAL_POSITION"
	var columns []models.Column
	if err := s.DB.Select(&columns, sql, s.DBName, tableName); err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}

	return columns, nil
}

func (s *MySQLSchema) buildColumnSQL(action string, columns []models.Column, col models.Column) string {
	extra := strings.Replace(col.Extra.String, "DEFAULT_GENERATED", "", -1)
	nullable := "NOT NULL"
	if col.IsNullable == "YES" {
		nullable = "NULL"
	}

	defaultValue := ""
	if col.ColumnDefault.Valid {
		if utils.IsNumericType(col.DataType.String) || col.ColumnDefault.String == "CURRENT_TIMESTAMP" {
			defaultValue = "DEFAULT " + col.ColumnDefault.String
		} else {
			defaultValue = "DEFAULT '" + col.ColumnDefault.String + "'"
		}
	}

	comment := ""
	if col.ColumnComment.Valid {
		comment = "COMMENT '" + col.ColumnComment.String + "'"
	}

	position := "FIRST"
	if col.OrdinalPosition > 1 {
		afterColumn := columns[col.OrdinalPosition-2].ColumnName.String
		position = "AFTER " + utils.EscapeIdentifier(afterColumn)
	}

	sql := fmt.Sprintf(
		"ALTER TABLE %s %s COLUMN %s %s %s %s %s %s %s;",
		utils.EscapeIdentifier(col.TableName.String),
		action,
		utils.EscapeIdentifier(col.ColumnName.String),
		col.ColumnType,
		extra,
		nullable,
		defaultValue,
		comment,
		position,
	)

	return sql
}

func (s *MySQLSchema) CreateAddColumn(columns []models.Column, col models.Column) string {
	return s.buildColumnSQL("ADD", columns, col)
}

func (s *MySQLSchema) CreateModifyColumn(columns []models.Column, col models.Column) string {
	return s.buildColumnSQL("MODIFY", columns, col)
}

func (s *MySQLSchema) CreateDropColumn(tableName, columnName string) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s DROP COLUMN %s;",
		utils.EscapeIdentifier(tableName),
		utils.EscapeIdentifier(columnName),
	)

	return sql
}

func (s *MySQLSchema) GetIndexes(tableName string) ([]models.IndexInfo, error) {
	sql := `
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) AS COLUMNS,
			NON_UNIQUE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY TABLE_NAME, INDEX_NAME, NON_UNIQUE
	`

	var indexes []models.IndexInfo
	err := s.DB.Select(&indexes, sql, s.DBName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
	}

	return indexes, nil
}

func (s *MySQLSchema) CreateAddIndex(index models.IndexInfo) string {
	indexType := "INDEX"
	if index.IndexName == "PRIMARY" {
		indexType = "PRIMARY KEY"
	} else if index.NonUnique == 0 {
		indexType = "UNIQUE INDEX"
	}

	cols := strings.Split(index.Columns, ",")
	for i, c := range cols {
		cols[i] = utils.EscapeIdentifier(c)
	}
	escapedCols := strings.Join(cols, ",")

	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD %s %s (%s);",
		utils.EscapeIdentifier(index.TableName),
		indexType,
		utils.EscapeIdentifier(index.IndexName),
		escapedCols,
	)

	return sql
}

func (s *MySQLSchema) CreateDropIndex(tableName, indexName string) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s DROP INDEX %s;",
		utils.EscapeIdentifier(tableName),
		utils.EscapeIdentifier(indexName),
	)
	return sql
}

func (s *MySQLSchema) GetForeignKeys(tableName string) ([]models.ForeignKey, error) {
	sql := `
		SELECT
			k.CONSTRAINT_NAME,
			k.TABLE_NAME,
			k.COLUMN_NAME,
			k.REFERENCED_TABLE_NAME,
			k.REFERENCED_COLUMN_NAME,
			rc.UPDATE_RULE,
			rc.DELETE_RULE
		FROM information_schema.KEY_COLUMN_USAGE k
		JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON k.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND k.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE k.TABLE_SCHEMA = ?
		  AND k.REFERENCED_TABLE_NAME IS NOT NULL
		  AND k.TABLE_NAME = ?
		ORDER BY k.CONSTRAINT_NAME, k.ORDINAL_POSITION
	`

	var foreignKeys []models.ForeignKey
	err := s.DB.Select(&foreignKeys, sql, s.DBName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", tableName, err)
	}

	return foreignKeys, nil
}

func (s *MySQLSchema) CreateForeignKey(fk models.ForeignKey) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE %s ON DELETE %s;",
		utils.EscapeIdentifier(fk.TableName),
		utils.EscapeIdentifier(fk.ConstraintName),
		utils.EscapeIdentifier(fk.ColumnName),
		utils.EscapeIdentifier(fk.ReferencedTable),
		utils.EscapeIdentifier(fk.ReferencedColumn),
		fk.UpdateRule,
		fk.DeleteRule,
	)

	return sql
}

func (s *MySQLSchema) CreateDropForeignKey(tableName, constraintName string) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s DROP FOREIGN KEY %s;",
		utils.EscapeIdentifier(tableName),
		utils.EscapeIdentifier(constraintName),
	)
	return sql
}

func (s *MySQLSchema) CreateDropTable(tableName string) string {
	sql := fmt.Sprintf("DROP TABLE %s;", utils.EscapeIdentifier(tableName))
	return sql
}

func (s *MySQLSchema) Close() error {
	return s.DB.Close()
}
