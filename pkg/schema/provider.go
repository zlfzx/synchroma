package schema

import "synchroma/pkg/models"

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
	CreateAlterTableProperties(tableName string, props []string) string
	CreateForeignKey(fk models.ForeignKey) string
	CreateDropForeignKey(tableName, constraintName string) string
	CreateDropTable(tableName string) string

	GetViews() ([]models.SchemaObject, error)
	GetTriggers() ([]models.SchemaObject, error)
	GetRoutines() ([]models.SchemaObject, error)

	GetViewDefinition(name string) (string, error)
	GetTriggerDefinition(name string) (string, error)
	GetRoutineDefinition(name, routineType string) (string, error)

	CreateDropView(name string) string
	CreateDropTrigger(name string) string
	CreateDropRoutine(name, routineType string) string

	DisableFKChecks() string
	EnableFKChecks() string
	
	Close() error
}
