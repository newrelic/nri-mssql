package inventory

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/inventory"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
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
	// Uncomment the following line to enable logging during tests
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)
	tests := []struct {
		name               string
		spConfigSetup      func(mock sqlmock.Sqlmock) // Function to mock sp_configure query
		sysConfigSetup     func(mock sqlmock.Sqlmock) // Function to mock sys.configurations query
		expectedInventory  map[string]inventory.Item  // Expected inventory items
		engineEditionValue int                        // Engine edition value
	}{
		{
			name: "Successful inventory population",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				spConfigRows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
					AddRow("allow polybase export", 0, 1, 0, 0).
					AddRow("allow updates", 0, 1, 0, 0)
				mock.ExpectQuery(spConfigQuery).WillReturnRows(spConfigRows)
			},
			sysConfigSetup: func(mock sqlmock.Sqlmock) {
				sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
					AddRow("allow polybase export", 1).
					AddRow("allow updates", 1)
				mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)
			},
			expectedInventory: map[string]inventory.Item{
				"allow polybase export/run_value":    {"value": 0},
				"allow updates/run_value":            {"value": 0},
				"allow polybase export/config_value": {"value": 1},
				"allow updates/config_value":         {"value": 1},
			},
			engineEditionValue: 3,
		},
		{
			name: "Engine edition 5: Only sys.configurations items collected",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				// No rows/queries expected for sp_configure since it should be skipped
			},
			sysConfigSetup: func(mock sqlmock.Sqlmock) {
				sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
					AddRow("allow polybase export", 1).
					AddRow("allow updates", 1)
				mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)
			},
			expectedInventory: map[string]inventory.Item{
				"allow polybase export/config_value": {"value": 1},
				"allow updates/config_value":         {"value": 1},
			},
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber, // Azure SQL Database
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := integration.New("test", "1.0.0")
			if err != nil {
				t.Fatalf("Unexpected error %s", err.Error())
			}

			e, err := i.Entity("test", "instance")
			if err != nil {
				t.Fatalf("Unexpected error %s", err.Error())
			}

			conn, mock := CreateMockSQL(t)

			// Setup mocks
			tt.spConfigSetup(mock)
			tt.sysConfigSetup(mock)

			PopulateInventory(e, conn, tt.engineEditionValue)

			// Validate inventory
			equal := true
			for k, expectedV := range tt.expectedInventory {
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
				t.Errorf("Expected %+v got %+v", tt.expectedInventory, e.Inventory.Items())
			}
		})
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

	engineEdition := 3

	// SPConfig expect
	mock.ExpectQuery(spConfigQuery).WillReturnError(errors.New("error"))

	// sys.configurations expect
	sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
		AddRow("allow polybase export", 1).
		AddRow("allow updates", 1)
	mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)

	PopulateInventory(e, conn, engineEdition)

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

	engineEdition := 3

	// SPConfig expect
	spConfigRows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
		AddRow("allow polybase export", 0, 1, 0, 0).
		AddRow("allow updates", 0, 1, 0, 0)
	mock.ExpectQuery(spConfigQuery).WillReturnRows(spConfigRows)

	// sys.configurations expect
	mock.ExpectQuery(sysConfigQuery).WillReturnError(errors.New("error"))

	PopulateInventory(e, conn, engineEdition)

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
