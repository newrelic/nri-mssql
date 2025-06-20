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

var errForTesting = errors.New("error")

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

type InventoryTestCase struct {
	name               string
	spConfigSetup      func(mock sqlmock.Sqlmock) // Function to mock sp_configure query
	sysConfigSetup     func(mock sqlmock.Sqlmock) // Function to mock sys.configurations query
	expectedInventory  map[string]inventory.Item  // Expected inventory items
	engineEditionValue int                        // Engine edition value
}

func runTestPopulateInventory(t *testing.T, tt InventoryTestCase) {
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

func Test_populateInventory_SuccessCases(t *testing.T) {
	// Uncomment the following line to enable logging during tests
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)
	tests := []InventoryTestCase{
		{
			name: "Successful inventory population for regular sql server",
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
			name: "Successful inventory collection for azure sql database",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				spConfigRows := sqlmock.NewRows([]string{"name", "value_in_use"}).
					AddRow("allow polybase export", 2).
					AddRow("allow updates", 2)
				mock.ExpectQuery(spConfigQueryForAzureSQLDatabase).WillReturnRows(spConfigRows)
			},
			sysConfigSetup: func(mock sqlmock.Sqlmock) {
				sysConfigRows := sqlmock.NewRows([]string{"name", "value"}).
					AddRow("allow polybase export", 1).
					AddRow("allow updates", 1)
				mock.ExpectQuery(sysConfigQuery).WillReturnRows(sysConfigRows)
			},
			expectedInventory: map[string]inventory.Item{
				"allow polybase export/run_value":    {"value": 2},
				"allow updates/run_value":            {"value": 2},
				"allow polybase export/config_value": {"value": 1},
				"allow updates/config_value":         {"value": 1},
			},
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber, // Azure SQL Database
		},
	}
	for _, tt := range tests {
		runTestPopulateInventory(t, tt)
	}
}

func Test_populateInventory_SPConfigErrorCases(t *testing.T) {
	// Uncomment the following line to enable logging during tests
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)
	tests := []InventoryTestCase{
		{
			name: "Error spConfig and success sysConfig for regular sql server",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(spConfigQuery).WillReturnError(errForTesting)
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
			engineEditionValue: 3,
		},
		{
			name: "Error spConfig and success sysConfig for azure sql database",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(spConfigQueryForAzureSQLDatabase).WillReturnError(errForTesting)
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
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber,
		},
	}
	for _, tt := range tests {
		runTestPopulateInventory(t, tt)
	}
}

func Test_populateInventory_SysConfigErrorCases(t *testing.T) {
	// Uncomment the following line to enable logging during tests
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)
	tests := []InventoryTestCase{
		{
			name: "Error sysConfig and success spConfig for regular sql server",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				spConfigRows := sqlmock.NewRows([]string{"name", "minimum", "maximum", "config_value", "run_value"}).
					AddRow("allow polybase export", 0, 1, 0, 0).
					AddRow("allow updates", 0, 1, 0, 0)
				mock.ExpectQuery(spConfigQuery).WillReturnRows(spConfigRows)
			},
			sysConfigSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sysConfigQuery).WillReturnError(errForTesting)
			},
			expectedInventory: map[string]inventory.Item{
				"allow polybase export/run_value": {"value": 0},
				"allow updates/run_value":         {"value": 0},
			},
			engineEditionValue: 3,
		},
		{
			name: "Error sysConfig and success spConfig for azure sql database",
			spConfigSetup: func(mock sqlmock.Sqlmock) {
				spConfigRows := sqlmock.NewRows([]string{"name", "value_in_use"}).
					AddRow("allow polybase export", 2).
					AddRow("allow updates", 2)
				mock.ExpectQuery(spConfigQueryForAzureSQLDatabase).WillReturnRows(spConfigRows)
			},
			sysConfigSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(sysConfigQuery).WillReturnError(errForTesting)
			},
			expectedInventory: map[string]inventory.Item{
				"allow polybase export/run_value": {"value": 2},
				"allow updates/run_value":         {"value": 2},
			},
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber, // Azure SQL Database
		},
	}

	for _, tt := range tests {
		runTestPopulateInventory(t, tt)
	}
}
