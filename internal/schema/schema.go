package schema

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"synchroma/internal/models"
	"synchroma/internal/utils"

	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Schema struct {
	DB     *sqlx.DB
	DBName string
	Tables []models.Table
}

func InitSchema(config models.DataSource) Schema {

	if config.Database == "" || config.Database != "mysql" {
		fmt.Println("database not supported")
		os.Exit(1)
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
		log.Fatal(err)
	}

	return Schema{
		DB:     DBSource,
		DBName: config.DBName,
	}
}

func (s *Schema) GetTableDependencies() (tableDependencies map[string][]string) {
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
		log.Fatal(err)
	}

	tableDependencies = make(map[string][]string)
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
	for _, t := range s.GetTables() {
		if _, ok := tableDependencies[t.TableName.String]; !ok {
			tableDependencies[t.TableName.String] = []string{}
		}
	}

	return
}

func (s *Schema) GetTables() (tables []models.Table) {
	if s.Tables != nil {
		return s.Tables
	}

	sql := "SELECT * FROM information_schema.tables WHERE table_schema = ?"
	err := s.DB.Select(&tables, sql, s.DBName)
	if err != nil {
		log.Fatal(err)
	}

	s.Tables = tables

	return
}

func (s *Schema) CreateTable(tableName string) (createTable string) {
	var ddl models.CreateTable

	sql := "SHOW CREATE TABLE " + tableName
	if err := s.DB.Get(&ddl, sql); err != nil {
		log.Fatal(err)
	}

	createTable = ddl.CreateTable

	// remove auto increment
	re := regexp.MustCompile(` AUTO_INCREMENT=\d+`)
	createTable = re.ReplaceAllString(createTable, "")

	createTable += ";"

	return
}

func (s *Schema) GetColumns(tableName string) (columns []models.Column) {
	sql := "SELECT * FROM information_schema.columns WHERE table_schema = ? AND table_name = ?"
	if err := s.DB.Select(&columns, sql, s.DBName, tableName); err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Schema) CreateAddColumn(columns []models.Column, col models.Column) string {
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
		position = "AFTER " + afterColumn
	}

	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN %s %s %s %s %s %s %s;",
		col.TableName.String,
		col.ColumnName.String,
		col.ColumnType,
		extra,
		nullable,
		defaultValue,
		comment,
		position,
	)

	return sql
}

func (s *Schema) CreateModifyColumn(columns []models.Column, col models.Column) string {
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
		position = "AFTER " + afterColumn
	}

	sql := fmt.Sprintf(
		"ALTER TABLE %s MODIFY COLUMN %s %s %s %s %s %s %s;",
		col.TableName.String,
		col.ColumnName.String,
		col.ColumnType,
		extra,
		nullable,
		defaultValue,
		comment,
		position,
	)

	return sql
}

func (s *Schema) CreateDropColumn(tableName, columnName string) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s DROP COLUMN `%s`;",
		tableName,
		columnName,
	)

	return sql
}

func (s *Schema) GetIndexes(tableName string) (indexes []models.IndexInfo) {
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

	err := s.DB.Select(&indexes, sql, s.DBName, tableName)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Schema) CreateAddIndex(index models.IndexInfo) string {
	indexType := "INDEX"
	if index.IndexName == "PRIMARY" {
		indexType = "PRIMARY KEY"
	} else if index.NonUnique == 0 {
		indexType = "UNIQUE INDEX"
	}

	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD %s %s (%s);",
		index.TableName,
		indexType,
		utils.EscapeIdentifier(index.IndexName),
		index.Columns,
	)

	return sql
}

func (s *Schema) GetForeignKeys(tableName string) (foreignKeys []models.ForeignKey) {
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

	err := s.DB.Select(&foreignKeys, sql, s.DBName, tableName)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Schema) CreateForeignKey(fk models.ForeignKey) string {
	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`) ON UPDATE %s ON DELETE %s;",
		fk.TableName,
		fk.ConstraintName,
		fk.ColumnName,
		fk.ReferencedTable,
		fk.ReferencedColumn,
		fk.UpdateRule,
		fk.DeleteRule,
	)

	return sql
}

func (s *Schema) Close() {
	if err := s.DB.Close(); err != nil {
		log.Fatal(err)
	}
}
