// Package instance contains helper methods for instance-level metric collection
package instance

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
)

// instanceNameQuery gets the instance name
const instanceNameQuery = "select COALESCE( @@SERVERNAME, SERVERPROPERTY('ServerName'), SERVERPROPERTY('MachineName')) as instance_name"

var ErrExpectedOneRow = errors.New("expected 1 row for instance name")

// NameRow is a row result in the instanceNameQuery
type NameRow struct {
	Name sql.NullString `db:"instance_name"`
}

// CreateInstanceEntity runs a query to get the instance
func CreateInstanceEntity(i *integration.Integration, con *connection.SQLConnection) (*integration.Entity, error) {
	instanceRows := make([]*NameRow, 0)
	if err := con.Query(&instanceRows, instanceNameQuery); err != nil {
		return nil, err
	}

	if length := len(instanceRows); length != 1 {
		return nil, fmt.Errorf("%w, but got %d", ErrExpectedOneRow, length)
	}

	if instanceRows[0].Name.Valid {
		instanceNameIDAttr := integration.NewIDAttribute("instance", instanceRows[0].Name.String)
		return i.EntityReportedVia(con.Host, instanceRows[0].Name.String, "ms-instance", instanceNameIDAttr)
	}

	return i.EntityReportedVia(con.Host, con.Host, "ms-instance")
}
