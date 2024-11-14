package models

type QueryConfig struct {
	Name  string `json:"name"`
	Query string `json:"query"`
	Type  string `json:"type"`
}
