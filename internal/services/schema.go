package services

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"synchroma/internal/models"

	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Schema struct {
	DB     *sqlx.DB
	DBName string
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

func (s *Schema) GetTables() (tables []models.Table) {
	sql := "SELECT * FROM information_schema.tables WHERE table_schema = ?"
	err := s.DB.Select(&tables, sql, s.DBName)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Schema) CreateTable(tableName string) (createTable string) {
	var ddl models.CreateTable

	sql := "SHOW CREATE TABLE " + tableName
	err := s.DB.Get(&ddl, sql)
	if err != nil {
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
	err := s.DB.Select(&columns, sql, s.DBName, tableName)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Schema) CreateColumn(tableName string, sourceColumns []models.Column, diffColumns map[string]models.Column) (alterColumn string) {
	for _, v := range diffColumns {

		extra := v.Extra.String
		extra = strings.Replace(extra, "DEFAULT_GENERATED", "", -1)

		nullable := "NOT NULL"
		if v.IsNullable == "YES" {
			nullable = "NULL"
		}

		default_value := ""
		if v.ColumnDefault.String != "" {
			if strings.Contains(v.ColumnType, "int") ||
				strings.Contains(v.ColumnType, "float") ||
				strings.Contains(v.ColumnType, "double") ||
				strings.Contains(v.ColumnType, "bool") ||
				v.ColumnDefault.String == "CURRENT_TIMESTAMP" {
				default_value = "DEFAULT " + v.ColumnDefault.String
			} else {
				default_value = "DEFAULT '" + v.ColumnDefault.String + "'"
			}

		}

		comment := ""
		if v.ColumnComment.String != "" {
			comment = "COMMENT '" + v.ColumnComment.String + "'"
		}

		position := "FIRST"
		if v.OrdinalPosition > 1 {
			afterColumn := sourceColumns[v.OrdinalPosition-2].ColumnName.String
			position = "AFTER " + afterColumn
		}

		// ALTER TABLE :table_name ADD COLUMN :column_name :column_type :extra :is_nullable :default_value :comment :position;

		output := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s %s %s %s %s %s;",
			tableName,
			v.ColumnName.String,
			v.ColumnType,
			extra,
			nullable,
			default_value,
			comment,
			position,
		)

		// remove multiple spaces
		re := regexp.MustCompile(`\s+`)
		output = strings.TrimSpace(re.ReplaceAllString(output, " "))
		output += "\n"

		alterColumn += output
	}

	return
}
