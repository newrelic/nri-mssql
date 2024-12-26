package validation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestValidatePreConditions(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock GetDatabaseDetails
	mock.ExpectQuery("select name, compatibility_level, is_query_store_on from sys.databases").WillReturnRows(
		sqlmock.NewRows([]string{"name", "compatibility_level", "is_query_store_on"}).
			AddRow("TestDB", 100, true),
	)

	// Mock checkPermissions
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := ValidatePreConditions(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_ErrorGettingDatabaseDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock GetDatabaseDetails error
	mock.ExpectQuery("select name, compatibility_level, is_query_store_on from sys.databases").WillReturnError(errors.New("query error"))

	result := ValidatePreConditions(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckDatabaseCompatibility(t *testing.T) {
	databaseDetails := []models.DatabaseDetailsDto{
		{Name: "TestDB1", Compatibility: 100},
		{Name: "TestDB2", Compatibility: 80},
	}

	result := checkDatabaseCompatibility(databaseDetails)
	assert.False(t, result)
}

func TestCheckQueryStores(t *testing.T) {
	databaseDetails := []models.DatabaseDetailsDto{
		{Name: "TestDB1", IsQueryStoreOn: true},
		{Name: "TestDB2", IsQueryStoreOn: false},
	}

	result := checkQueryStores(databaseDetails)
	assert.True(t, result)
}

func TestCheckPermissionsAndLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkPermissions
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckQueryStores_AllQueryStoresOff(t *testing.T) {
	databaseDetails := []models.DatabaseDetailsDto{
		{Name: "TestDB1", IsQueryStoreOn: false},
		{Name: "TestDB2", IsQueryStoreOn: false},
	}

	result := checkQueryStores(databaseDetails)
	assert.False(t, result)
}

func TestCheckPermissionsAndLogin_PermissionsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkPermissions error
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnError(fmt.Errorf("query error"))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_MissingPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkPermissions returning false
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(false))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_LoginEnabledError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkSQLServerLoginEnabled error
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnError(fmt.Errorf("query error"))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_LoginNotEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkSQLServerLoginEnabled returning false
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(false))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_LoginEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Mock checkPermissions
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled returning true
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
