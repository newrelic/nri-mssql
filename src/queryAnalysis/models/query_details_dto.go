package models

type QueryDetailsDto struct {
	Name           string `json:"name"`
	Query          string `json:"query"`
	Type           string `json:"type"`
	ResponseDetail interface{}
}
