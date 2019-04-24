// Package instance contains helper methods for instance-level metric collection
package instance

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
)

// instanceNameQuery gets the instance name
const instanceNameQuery = "select @@SERVERNAME as instance_name"

// NameRow is a row result in the instanceNameQuery
type NameRow struct {
	Name string `db:"instance_name"`
}

// CreateInstanceEntity runs a query to get the instance
func CreateInstanceEntity(i *integration.Integration, con *connection.SQLConnection) (*integration.Entity, error) {
	instanceRows := make([]*NameRow, 0)
	if err := con.Query(&instanceRows, instanceNameQuery); err != nil {
		return nil, err
	}

	if length := len(instanceRows); length != 1 {
		return nil, fmt.Errorf("expected 1 row for instance name got %d", length)
	}


  instanceNameIDAttr := integration.NewIDAttribute("instance", instanceRows[0].Name)
	return i.EntityReportedVia(con.Host, con.Host, "ms-instance", instanceNameIDAttr)
}
