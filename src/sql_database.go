package main

import (
	"github.com/newrelic/infra-integrations-sdk/integration"
)

// databaseNameQuery gets all database names
const databaseNameQuery = "select db_name from sys.databases"

// DatabaseNameRow is a row result in the databaseNameQuery
type DatabaseNameRow struct {
	DBName string `db:"db_name"`
}

func createDatabaseEntities(i *integration.Integration, con *SQLConnection) ([]*integration.Entity, error) {
	databaseRows := make([]*DatabaseNameRow, 0)
	if err := con.Query(&databaseRows, databaseNameQuery); err != nil {
		return nil, err
	}

	dbEntities := make([]*integration.Entity, 0, len(databaseRows))
	for _, row := range databaseRows {
		dbEntity, err := i.Entity(row.DBName, "database")
		if err != nil {
			return nil, err
		}

		dbEntities = append(dbEntities, dbEntity)
	}

	return dbEntities, nil
}
