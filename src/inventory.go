package main

import (
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// spConfigQuery query for inventory items
const spConfigQuery = "EXEC sp_configure"

// SPConfigRow represents a row in the table returned by spConfigQuery
type SPConfigRow struct {
	Name        string `db:"name"`
	Minimum     int    `db:"minimum"`
	Maximum     int    `db:"maximum"`
	ConfigValue int    `db:"config_value"`
	RunValue    int    `db:"run_value"`
}

//populateInventory runs spConfigQuery and populates the return values into the entity
func populateInventory(instanceEntity *integration.Entity, con *SQLConnection) error {
	configRows := make([]*SPConfigRow, 0)
	if err := con.Query(&configRows, spConfigQuery); err != nil {
		return err
	}

	for _, row := range configRows {
		setItemOrLog(instanceEntity, row.Name+"/minimum", row.Minimum)
		setItemOrLog(instanceEntity, row.Name+"/maximum", row.Maximum)
		setItemOrLog(instanceEntity, row.Name+"/config_value", row.ConfigValue)
		setItemOrLog(instanceEntity, row.Name+"/run_value", row.RunValue)
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
