package util

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/integration"
)

// instanceNameQuery gets the instance name
const instanceNameQuery = "select @@SERVERNAME as instance_name"

// InstanceNameRow is a row result in the instanceNameQuery
type InstanceNameRow struct {
	Name string `db:"instance_name"`
}

// CreateInstanceEntity runs a query to get the instance
func CreateInstanceEntity(i *integration.Integration, con *SQLConnection) (*integration.Entity, error) {
	instaceRows := make([]*InstanceNameRow, 0)
	if err := con.Query(&instaceRows, instanceNameQuery); err != nil {
		return nil, err
	}

	if length := len(instaceRows); length != 1 {
		return nil, fmt.Errorf("expected 1 row for instance name got %d", length)
	}

	return i.Entity(instaceRows[0].Name, "instance")
}
