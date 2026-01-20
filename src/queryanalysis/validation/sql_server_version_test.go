package validation

import (
	"regexp"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckSqlServerVersion_SupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a supported SQL Server version response
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2019 (RTM) - 15.0.2000.5"))

	result, _ := checkSQLServerVersion(sqlConnection, false)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSQLServerVersionforAzure(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"AzureSupportedVersion", "Microsoft SQL Azure (RTM) - 12.0.2000.8", true},
		{"AzureUnsupportedVersion", "Microsoft SQL azure (RTM) - 11.0.2000.7", false},
		{"AzureUnsupportedVersion", "Microsoft SQL (RTM) - 12.0.2000.8", false},
		{"AzureUnsupportedVersion", "Microsoft SQL Azure (RTM) - 17.0.2000.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlConnection, mock := setupMockDB(t)
			defer sqlConnection.Connection.Close()

			// Mocking the SQL Server version response based on the test case
			mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
				WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(tt.version))

			result, err := checkSQLServerVersion(sqlConnection, false)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckSqlServerVersion_UnsupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking an unsupported SQL Server version response
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2014 - 12.0.2000.8"))

	result, _ := checkSQLServerVersion(sqlConnection, false)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckSqlServerVersion_EmptyVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(""))
	result, _ := checkSQLServerVersion(sqlConnection, false)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_InvalidVersionString(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a malformed version string
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server  - version unknown"))

	result, _ := checkSQLServerVersion(sqlConnection, false)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckSqlServerVersion_QueryError(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).WillReturnError(errQueryError)
	result, _ := checkSQLServerVersion(sqlConnection, false)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSQLServerVersion(t *testing.T) {
	version, err := parseSQLServerVersion("Microsoft SQL Server 2019 (RTM) - 15.0.2000.5")
	assert.NoError(t, err)
	if version.Major > uint64(^uint(0)>>1) {
		t.Fatalf("version.Major value %d is too large to convert to int", version.Major)
	}
	assert.Equal(t, 15, int(version.Major))
}

func TestGetSQLServerVersion_EmptyVersion(t *testing.T) {
	_, err := parseSQLServerVersion("")
	assert.Error(t, err)
}

func TestGetSQLServerVersion_InvalidVersionString(t *testing.T) {
	_, err := parseSQLServerVersion("Microsoft SQL Server  - version unknown")
	assert.Error(t, err)
}

func TestGetSQLServerVersion_ParseError(t *testing.T) {
	_, err := parseSQLServerVersion("Invalid SQL Server version string")
	assert.Error(t, err)
}

// New tests for SQL Server 2016 and DMV-only mode
func TestCheckSqlServerVersion_SQL2016_DMVOnlyMode(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking SQL Server 2016 version
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2016 (RTM) - 13.0.1601.5"))

	result, err := checkSQLServerVersion(sqlConnection, true)
	assert.NoError(t, err)
	assert.True(t, result, "SQL Server 2016 should be supported in DMV-only mode")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_SQL2016_QueryStoreMode(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking SQL Server 2016 version
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2016 (RTM) - 13.0.1601.5"))

	result, err := checkSQLServerVersion(sqlConnection, false)
	assert.NoError(t, err)
	assert.False(t, result, "SQL Server 2016 should NOT be supported in Query Store mode")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_VersionBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		isDMVOnlyMode bool
		expected      bool
		description   string
	}{
		{
			name:          "SQL2014_QueryStoreMode",
			version:       "Microsoft SQL Server 2014 - 12.0.2000.8",
			isDMVOnlyMode: false,
			expected:      false,
			description:   "SQL Server 2014 should be rejected in Query Store mode",
		},
		{
			name:          "SQL2014_DMVOnlyMode",
			version:       "Microsoft SQL Server 2014 - 12.0.2000.8",
			isDMVOnlyMode: true,
			expected:      false,
			description:   "SQL Server 2014 should be rejected in DMV-only mode",
		},
		{
			name:          "SQL2016_QueryStoreMode",
			version:       "Microsoft SQL Server 2016 - 13.0.1601.5",
			isDMVOnlyMode: false,
			expected:      false,
			description:   "SQL Server 2016 should be rejected in Query Store mode",
		},
		{
			name:          "SQL2016_DMVOnlyMode",
			version:       "Microsoft SQL Server 2016 - 13.0.1601.5",
			isDMVOnlyMode: true,
			expected:      true,
			description:   "SQL Server 2016 should be accepted in DMV-only mode",
		},
		{
			name:          "SQL2017_QueryStoreMode",
			version:       "Microsoft SQL Server 2017 - 14.0.1000.169",
			isDMVOnlyMode: false,
			expected:      true,
			description:   "SQL Server 2017 should be accepted in Query Store mode",
		},
		{
			name:          "SQL2017_DMVOnlyMode",
			version:       "Microsoft SQL Server 2017 - 14.0.1000.169",
			isDMVOnlyMode: true,
			expected:      true,
			description:   "SQL Server 2017 should be accepted in DMV-only mode",
		},
		{
			name:          "SQL2022_QueryStoreMode",
			version:       "Microsoft SQL Server 2022 - 16.0.1000.6",
			isDMVOnlyMode: false,
			expected:      true,
			description:   "SQL Server 2022 should be accepted in Query Store mode",
		},
		{
			name:          "SQL2022_DMVOnlyMode",
			version:       "Microsoft SQL Server 2022 - 16.0.1000.6",
			isDMVOnlyMode: true,
			expected:      true,
			description:   "SQL Server 2022 should be accepted in DMV-only mode",
		},
		{
			name:          "FutureVersion_QueryStoreMode",
			version:       "Microsoft SQL Server 2025 - 17.0.1000.0",
			isDMVOnlyMode: false,
			expected:      false,
			description:   "Future SQL Server version should be rejected in Query Store mode",
		},
		{
			name:          "FutureVersion_DMVOnlyMode",
			version:       "Microsoft SQL Server 2025 - 17.0.1000.0",
			isDMVOnlyMode: true,
			expected:      false,
			description:   "Future SQL Server version should be rejected in DMV-only mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlConnection, mock := setupMockDB(t)
			defer sqlConnection.Connection.Close()

			mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
				WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(tt.version))

			result, err := checkSQLServerVersion(sqlConnection, tt.isDMVOnlyMode)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result, tt.description)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckSqlServerVersion_Azure_DMVOnlyMode(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		isDMVOnlyMode bool
		expected      bool
		description   string
	}{
		{
			name:          "AzureSQL_v12_QueryStoreMode",
			version:       "Microsoft SQL Azure (RTM) - 12.0.2000.8",
			isDMVOnlyMode: false,
			expected:      true,
			description:   "Azure SQL v12 should be accepted in Query Store mode",
		},
		{
			name:          "AzureSQL_v12_DMVOnlyMode",
			version:       "Microsoft SQL Azure (RTM) - 12.0.2000.8",
			isDMVOnlyMode: true,
			expected:      true,
			description:   "Azure SQL v12 should be accepted in DMV-only mode",
		},
		{
			name:          "AzureSQL_v16_QueryStoreMode",
			version:       "Microsoft SQL Azure (RTM) - 16.0.2000.8",
			isDMVOnlyMode: false,
			expected:      true,
			description:   "Azure SQL v16 should be accepted in Query Store mode",
		},
		{
			name:          "AzureSQL_v16_DMVOnlyMode",
			version:       "Microsoft SQL Azure (RTM) - 16.0.2000.8",
			isDMVOnlyMode: true,
			expected:      true,
			description:   "Azure SQL v16 should be accepted in DMV-only mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlConnection, mock := setupMockDB(t)
			defer sqlConnection.Connection.Close()

			mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
				WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(tt.version))

			result, err := checkSQLServerVersion(sqlConnection, tt.isDMVOnlyMode)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result, tt.description)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
