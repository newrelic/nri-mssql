package validation

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"
)

func setupMockDB(t *testing.T) (*connection.SQLConnection, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}
	return sqlConnection, mock
}

func mockCheckPermissions(mock sqlmock.Sqlmock, hasPermission bool) {
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(hasPermission))
}

func mockCheckSQLServerLoginEnabled(mock sqlmock.Sqlmock, isLoginEnabled bool) {
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(isLoginEnabled))
}
