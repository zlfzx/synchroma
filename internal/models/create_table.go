package models

type CreateTable struct {
	Table       string `db:"Table"`
	CreateTable string `db:"Create Table"`
}
