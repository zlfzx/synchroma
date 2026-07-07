package schema

import "synchroma/internal/models"

type SchemaProvider interface {
	GetTableDependencies() (map[string][]string, error)
	GetTables() ([]models.Table, error)
	CreateTable(tableName string) (string, error)
	GetColumns(tableName string) ([]models.Column, error)
	GetIndexes(tableName string) ([]models.IndexInfo, error)
	GetForeignKeys(tableName string) ([]models.ForeignKey, error)
	
	CreateAddColumn(columns []models.Column, col models.Column) string
	CreateModifyColumn(columns []models.Column, col models.Column) string
	CreateDropColumn(tableName, columnName string) string
	CreateAddIndex(index models.IndexInfo) string
	CreateDropIndex(tableName, indexName string) string
	CreateForeignKey(fk models.ForeignKey) string
	CreateDropForeignKey(tableName, constraintName string) string
	CreateDropTable(tableName string) string
	
	Close() error
}
