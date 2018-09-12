package inventory

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// CreateMockSQL creates a Test SQLConnection.
func CreateMockSQL(t *testing.T) (con *connection.SQLConnection, mock sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Unexpected error while mocking: %s", err.Error())
		t.FailNow()
	}

	con = &connection.SQLConnection{
		Connection: sqlx.NewDb(mockDB, "sqlmock"),
	}

	return
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

	conn, mock := CreateMockSQL(t)

	// SPConfig expect
	spConfigRows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
		AddRow("allow polybase export", 0, 1, 0, 0).
		AddRow("allow updates", 0, 1, 0, 0)
	mock.ExpectQuery(spConfigQuery).WillReturnRows(spConfigRows)

	// sys.configurations expect
	sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
		AddRow("allow polybase export", 1).
		AddRow("allow updates", 1)
	mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)

	PopulateInventory(e, conn)

	// expected inventory.Items map to be set
	expected := map[string]inventory.Item{
		"allow polybase export/run_value":    {"value": 0},
		"allow updates/run_value":            {"value": 0},
		"allow polybase export/config_value": {"value": 1},
		"allow updates/config_value":         {"value": 1},
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

func Test_populateInventory_SPConfigError(t *testing.T) {
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

	conn, mock := CreateMockSQL(t)

	// SPConfig expect
	mock.ExpectQuery(spConfigQuery).WillReturnError(errors.New("error"))

	// sys.configurations expect
	sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
		AddRow("allow polybase export", 1).
		AddRow("allow updates", 1)
	mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)

	PopulateInventory(e, conn)

	// expected inventory.Items map to be set
	expected := map[string]inventory.Item{
		"allow polybase export/config_value": {"value": 1},
		"allow updates/config_value":         {"value": 1},
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

func Test_populateInventory_SysConfigError(t *testing.T) {
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

	conn, mock := CreateMockSQL(t)

	// SPConfig expect
	spConfigRows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
		AddRow("allow polybase export", 0, 1, 0, 0).
		AddRow("allow updates", 0, 1, 0, 0)
	mock.ExpectQuery(spConfigQuery).WillReturnRows(spConfigRows)

	// sys.configurations expect
	mock.ExpectQuery(sysConfigQuery).WillReturnError(errors.New("error"))

	PopulateInventory(e, conn)

	// expected inventory.Items map to be set
	expected := map[string]inventory.Item{
		"allow polybase export/run_value": {"value": 0},
		"allow updates/run_value":         {"value": 0},
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
