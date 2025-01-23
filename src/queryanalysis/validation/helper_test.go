package validation

import (
	"errors"
	"regexp"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// constants
var errQueryError = errors.New("query error")

// util functions
func setupMockDB(t *testing.T) (*connection.SQLConnection, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}
	return sqlConnection, mock
}

func mockCheckPermissions(mock sqlmock.Sqlmock, hasPermission bool) {
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(hasPermission))
}

func mockCheckSQLServerLoginEnabled(mock sqlmock.Sqlmock, isLoginEnabled bool) {
	mock.ExpectQuery(regexp.QuoteMeta(checkSQLServerLoginEnabledQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(isLoginEnabled))
}
