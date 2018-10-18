// Package inventory contains all the code used to collect inventory items from the target
package inventory

import (
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/connection"
)

const (
	spConfigQuery  = "EXEC sp_configure"
	sysConfigQuery = "select name, value from sys.configurations"
)

// SPConfigRow represents a row in the table returned by spConfigQuery
type SPConfigRow struct {
	Name        string `db:"name"`
	Minimum     int    `db:"minimum"`      // not used but needed in order to unmarshal from query results
	Maximum     int    `db:"maximum"`      // not used but needed in order to unmarshal from query results
	ConfigValue int    `db:"config_value"` // not used but needed in order to unmarshal from query results
	RunValue    int    `db:"run_value"`
}

// ConfigQueryRow represents a row in the table returned by sysConfigQuery
type ConfigQueryRow struct {
	Name  string `db:"name"`
	Value int    `db:"value"`
}

// PopulateInventory gathers inventory data for the SQL Server instance and populates it into entity
func PopulateInventory(instanceEntity *integration.Entity, connection *connection.SQLConnection) {
	if err := populateSPConfigItems(instanceEntity, connection); err != nil {
		log.Error("Error collecting inventory items from sp_config: %s", err.Error())
	}

	if err := populateSysConfigItems(instanceEntity, connection); err != nil {
		log.Error("Error collecting inventory items from sys.configurations: %s", err.Error())
	}
}

// populateSPConfigItems collects inventory items for sp_configure procedure
func populateSPConfigItems(instanceEntity *integration.Entity, connection *connection.SQLConnection) error {
	configRows := make([]*SPConfigRow, 0)
	if err := connection.Query(&configRows, spConfigQuery); err != nil {
		return err
	}

	for _, row := range configRows {
		itemName := row.Name + "/run_value"
		setItemOrLog(instanceEntity, itemName, row.RunValue)
	}

	return nil
}

// populateSysConfigItems collect inventory items from sys.configurations
func populateSysConfigItems(instanceEntity *integration.Entity, connection *connection.SQLConnection) error {
	configRows := make([]*ConfigQueryRow, 0)
	if err := connection.Query(&configRows, sysConfigQuery); err != nil {
		return err
	}

	for _, row := range configRows {
		itemName := row.Name + "/config_value"
		setItemOrLog(instanceEntity, itemName, row.Value)
	}

	return nil
}

// setItemOrLog attempts to set and inventory item. If there
// is an error it is logged as such
func setItemOrLog(instanceEntity *integration.Entity, key string, value interface{}) {
	if err := instanceEntity.SetInventoryItem(key, "value", value); err != nil {
		log.Error("Error setting inventory item '%s': %s", key, err.Error())
	}
}
