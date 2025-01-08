package models

type DatabaseDetailsDto struct {
	Name           string `db:"name"`
	IsQueryStoreOn bool   `db:"is_query_store_on"`
	Compatibility  int    `db:"compatibility_level"`
}
