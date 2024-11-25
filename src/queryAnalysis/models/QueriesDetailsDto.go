package models

type QueryDetailsDto struct {
	Name            string
	Query           string
	ResultStructure map[string]string
}
