package models

type Index struct {
	Table        string `db:"Table"`
	NonUnique    int    `db:"Non_unique"`
	KeyName      string `db:"Key_name"`
	SeqInIndex   int    `db:"Seq_in_index"`
	ColumnName   string `db:"Column_name"`
	Collation    string `db:"Collation"`
	Cardinality  int    `db:"Cardinality"`
	SubPart      int    `db:"Sub_part"`
	Packed       string `db:"Packed"`
	Null         string `db:"Null"`
	IndexType    string `db:"Index_type"`
	Comment      string `db:"Comment"`
	IndexComment string `db:"Index_comment"`
	Visible      string `db:"Visible"`
	Expression   string `db:"Expression"`
}
