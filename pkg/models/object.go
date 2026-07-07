package models

type SchemaObject struct {
	Name string `db:"NAME"`
	Type string `db:"TYPE"`
}

type CreateView struct {
	View       string `db:"View"`
	CreateView string `db:"Create View"`
}

type CreateTrigger struct {
	Trigger              string `db:"Trigger"`
	SQLOriginalStatement string `db:"SQL Original Statement"`
}

type CreateProcedure struct {
	Procedure       string `db:"Procedure"`
	CreateProcedure string `db:"Create Procedure"`
}

type CreateFunction struct {
	Function       string `db:"Function"`
	CreateFunction string `db:"Create Function"`
}
