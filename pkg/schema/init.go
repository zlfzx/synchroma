package schema

import (
	"fmt"
	"synchroma/pkg/models"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func InitSchema(config models.DataSource) (SchemaProvider, error) {
	switch config.Database {
	case "mysql":
		return initMySQL(config)
	case "postgres", "postgresql":
		return initPostgres(config)
	default:
		return nil, fmt.Errorf("database '%s' is not supported. Supported: mysql, postgres", config.Database)
	}
}

func initMySQL(config models.DataSource) (SchemaProvider, error) {
	datasource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)

	db, err := sqlx.Connect("mysql", datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	return &MySQLSchema{
		DB:     db,
		DBName: config.DBName,
	}, nil
}

func initPostgres(config models.DataSource) (SchemaProvider, error) {
	sslMode := "disable"
	if config.Port == "" {
		config.Port = "5432"
	}

	datasource := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		sslMode,
	)

	db, err := sqlx.Connect("postgres", datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &PostgresSchema{
		DB:     db,
		DBName: config.DBName,
	}, nil
}
