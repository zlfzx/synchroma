package models

import "database/sql"

type Column struct {
	TableCatalog           sql.NullString `db:"TABLE_CATALOG"`
	TableSchema            sql.NullString `db:"TABLE_SCHEMA"`
	TableName              sql.NullString `db:"TABLE_NAME"`
	ColumnName             sql.NullString `db:"COLUMN_NAME"`
	OrdinalPosition        int            `db:"ORDINAL_POSITION"`
	ColumnDefault          sql.NullString `db:"COLUMN_DEFAULT"`
	IsNullable             string         `db:"IS_NULLABLE"`
	DataType               sql.NullString `db:"DATA_TYPE"`
	CharacterMaximumLength sql.NullInt64  `db:"CHARACTER_MAXIMUM_LENGTH"`
	CharacterOctetLength   sql.NullInt64  `db:"CHARACTER_OCTET_LENGTH"`
	NumericPrecision       sql.NullInt64  `db:"NUMERIC_PRECISION"`
	NumericScale           sql.NullInt64  `db:"NUMERIC_SCALE"`
	DatetimePrecision      sql.NullInt64  `db:"DATETIME_PRECISION"`
	CharacterSetName       sql.NullString `db:"CHARACTER_SET_NAME"`
	CollationName          sql.NullString `db:"COLLATION_NAME"`
	ColumnType             string         `db:"COLUMN_TYPE"`
	ColumnKey              string         `db:"COLUMN_KEY"`
	Extra                  sql.NullString `db:"EXTRA"`
	Privileges             sql.NullString `db:"PRIVILEGES"`
	ColumnComment          sql.NullString `db:"COLUMN_COMMENT"`
	GenerationExpression   sql.NullString `db:"GENERATION_EXPRESSION"`
	SrsId                  sql.NullInt64  `db:"SRS_ID"`
}
