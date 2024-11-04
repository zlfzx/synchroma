package models

import "database/sql"

type Table struct {
	TableCatalog   sql.NullString `db:"TABLE_CATALOG"`
	TableSchema    sql.NullString `db:"TABLE_SCHEMA"`
	TableName      sql.NullString `db:"TABLE_NAME"`
	TableType      string         `db:"TABLE_TYPE"`
	Engine         sql.NullString `db:"ENGINE"`
	Version        sql.NullInt64  `db:"VERSION"`
	RowFormat      sql.NullString `db:"ROW_FORMAT"`
	TableRows      sql.NullInt64  `db:"TABLE_ROWS"`
	AvgRowLength   sql.NullInt64  `db:"AVG_ROW_LENGTH"`
	DataLength     sql.NullInt64  `db:"DATA_LENGTH"`
	MaxDataLength  sql.NullInt64  `db:"MAX_DATA_LENGTH"`
	IndexLength    sql.NullInt64  `db:"INDEX_LENGTH"`
	DataFree       sql.NullInt64  `db:"DATA_FREE"`
	AutoIncrement  sql.NullInt64  `db:"AUTO_INCREMENT"`
	CreateTime     string         `db:"CREATE_TIME"`
	UpdateTime     sql.NullString `db:"UPDATE_TIME"`
	CheckTime      sql.NullString `db:"CHECK_TIME"`
	TableCollation sql.NullString `db:"TABLE_COLLATION"`
	CheckSum       sql.NullString `db:"CHECKSUM"`
	CreateOptions  sql.NullString `db:"CREATE_OPTIONS"`
	TableComment   sql.NullString `db:"TABLE_COMMENT"`
}
