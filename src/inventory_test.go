package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Test_populateInventory_QueryError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("test", "instance")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)
	defer conn.Close()

	mock.ExpectQuery(spConfigQuery).WillReturnError(errors.New("error"))

	if err := populateInventory(e, conn); err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_populateInventory(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("test", "instance")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)
	defer conn.Close()

	rows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
		AddRow("allow polybase export", 0, 1, 0, 0).
		AddRow("allow updates", 0, 1, 0, 0)
	mock.ExpectQuery(spConfigQuery).WillReturnRows(rows)

	if err := populateInventory(e, conn); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	// expected inventory.Items map to be set
	expected := map[string]inventory.Item{
		"allow polybase export/minimum":      {"value": 0},
		"allow polybase export/maximum":      {"value": 1},
		"allow polybase export/config_value": {"value": 0},
		"allow polybase export/run_value":    {"value": 0},
		"allow updates/minimum":              {"value": 0},
		"allow updates/maximum":              {"value": 1},
		"allow updates/config_value":         {"value": 0},
		"allow updates/run_value":            {"value": 0},
	}

	// reflect.DeepEqual did not work on comparing Items maps so did a manual comparison
	equal := true
	for k, expectedV := range expected {
		v, ok := e.Inventory.Item(k)
		if !ok {
			equal = false
			break
		}

		if !reflect.DeepEqual(expectedV, v) {
			equal = false
			break
		}
	}

	if !equal {
		t.Errorf("Expected %+v got %+v", expected, e.Inventory.Items())
	}
}
