package connection

import (
	"errors"
	"fmt"
	"testing"

	"github.com/newrelic/nri-mssql/src/args"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var ErrConnectionFailure = errors.New("something went wrong while trying to create the SQL connection")

func Test_SQLConnection_Close(t *testing.T) {
	conn, mock := CreateMockSQL(t)

	mock.ExpectClose().WillReturnError(fmt.Errorf("critical operation failed: %w", ErrConnectionFailure))
	conn.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("close expectation was not met: %s", err.Error())
	}
}

func Test_SQLConnection_Query(t *testing.T) {
	conn, mock := CreateMockSQL(t)

	// Temp data structure to store data into
	temp := []struct {
		One int `db:"one"`
		Two int `db:"two"`
	}{}

	// dummy query to run
	query := "select one, two from everywhere"

	rows := sqlmock.NewRows([]string{"one", "two"}).AddRow(1, 2)
	mock.ExpectQuery(query).WillReturnRows(rows)

	if err := conn.Query(&temp, query); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	if length := len(temp); length != 1 {
		t.Errorf("Expected 1 element got %d", length)
		t.FailNow()
	}

	if temp[0].One != 1 || temp[0].Two != 2 {
		t.Error("Query did not marshal correctly")
	}
}

func Test_createConnectionURL(t *testing.T) {
	testCases := []struct {
		name   string
		arg    *args.ArgumentList
		dbName string
		want   string
	}{
		{
			"Port No SSL",
			&args.ArgumentList{
				Username:  "user",
				Password:  "pass",
				Hostname:  "localhost",
				EnableSSL: false,
				Port:      "1443",
				Timeout:   "30",
			},
			"",
			"sqlserver://user:pass@localhost:1443?connection+timeout=30&dial+timeout=30",
		},
		{
			"Instance No SSL",
			&args.ArgumentList{
				Username:  "user",
				Password:  "pass",
				Hostname:  "localhost",
				EnableSSL: false,
				Instance:  "SQLExpress",
				Timeout:   "30",
			},
			"",
			"sqlserver://user:pass@localhost/SQLExpress?connection+timeout=30&dial+timeout=30",
		},
		{
			"Instance SSL Trust",
			&args.ArgumentList{
				Username:               "user",
				Password:               "pass",
				Hostname:               "localhost",
				EnableSSL:              true,
				TrustServerCertificate: true,
				Instance:               "SQLExpress",
				Timeout:                "30",
			},
			"",
			"sqlserver://user:pass@localhost/SQLExpress?TrustServerCertificate=true&connection+timeout=30&dial+timeout=30&encrypt=true",
		},
		{
			"Instance SSL Certificate",
			&args.ArgumentList{
				Username:               "user",
				Password:               "pass",
				Hostname:               "localhost",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "file.ca",
				Instance:               "SQLExpress",
				Timeout:                "30",
			},
			"",
			"sqlserver://user:pass@localhost/SQLExpress?TrustServerCertificate=false&certificate=file.ca&connection+timeout=30&dial+timeout=30&encrypt=true",
		},
		{
			"Extra Args",
			&args.ArgumentList{
				Username:               "user",
				Password:               "pass",
				Hostname:               "localhost",
				EnableSSL:              false,
				Port:                   "1443",
				Timeout:                "30",
				ExtraConnectionURLArgs: "applicationIntent=true",
			},
			"",
			"sqlserver://user:pass@localhost:1443?applicationIntent=true&connection+timeout=30&dial+timeout=30",
		},
		{
			"Database Name",
			&args.ArgumentList{
				Username:  "user",
				Password:  "pass",
				Hostname:  "localhost",
				EnableSSL: false,
				Port:      "1443",
				Timeout:   "30",
			},
			"test-db",
			"sqlserver://user:pass@localhost:1443?connection+timeout=30&database=test-db&dial+timeout=30",
		},
	}

	for _, tc := range testCases {
		if out := CreateConnectionURL(tc.arg, tc.dbName); out != tc.want {
			t.Errorf("Test Case %s Failed: Expected '%s' got '%s'", tc.name, tc.want, out)
		}
	}
}

func Test_CreateAzureADConnectionURL(t *testing.T) {
	testCases := []struct {
		name   string
		arg    *args.ArgumentList
		dbName string
		want   string
	}{
		{
			"Basic Service Principal No SSL",
			&args.ArgumentList{
				ClientID:     "12345678-1234-1234-1234-123456789012",
				TenantID:     "87654321-4321-4321-4321-210987654321",
				ClientSecret: "client-secret",
				Hostname:     "localhost",
				Port:         "1433",
				Timeout:      "30",
				EnableSSL:    false,
			},
			"",
			"server=localhost;port=1433;database=;user id=12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30",
		},
		{
			"Service Principal with Database",
			&args.ArgumentList{
				ClientID:     "abcdef12-3456-7890-abcd-ef1234567890",
				TenantID:     "fedcba09-8765-4321-fedc-ba0987654321",
				ClientSecret: "super-secret-key",
				Hostname:     "sqlserver.database.windows.net",
				Port:         "1433",
				Timeout:      "30",
				EnableSSL:    false,
			},
			"test-database",
			"server=sqlserver.database.windows.net;port=1433;database=test-database;user id=abcdef12-3456-7890-abcd-ef1234567890@fedcba09-8765-4321-fedc-ba0987654321;password=super-secret-key;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30",
		},
		{
			"Service Principal SSL Trust Certificate",
			&args.ArgumentList{
				ClientID:               "12345678-1234-1234-1234-123456789012",
				TenantID:               "87654321-4321-4321-4321-210987654321",
				ClientSecret:           "client-secret",
				Hostname:               "sqlserver.database.windows.net",
				Port:                   "1433",
				Timeout:                "30",
				EnableSSL:              true,
				TrustServerCertificate: true,
			},
			"production-db",
			"server=sqlserver.database.windows.net;port=1433;database=production-db;user id=12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30;encrypt=true;TrustServerCertificate=true",
		},
		{
			"Service Principal SSL with Certificate File",
			&args.ArgumentList{
				ClientID:               "12345678-1234-1234-1234-123456789012",
				TenantID:               "87654321-4321-4321-4321-210987654321",
				ClientSecret:           "client-secret",
				Hostname:               "sqlserver.database.windows.net",
				Port:                   "1433",
				Timeout:                "30",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "/path/to/cert.pem",
			},
			"secure-db",
			"server=sqlserver.database.windows.net;port=1433;database=secure-db;user id=12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30;encrypt=true;TrustServerCertificate=false;certificate=/path/to/cert.pem",
		},
		{
			"Service Principal SSL Don't Trust No Certificate",
			&args.ArgumentList{
				ClientID:               "12345678-1234-1234-1234-123456789012",
				TenantID:               "87654321-4321-4321-4321-210987654321",
				ClientSecret:           "client-secret",
				Hostname:               "localhost",
				Port:                   "1433",
				Timeout:                "60",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "",
			},
			"test-db",
			"server=localhost;port=1433;database=test-db;user id=12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=60;connection timeout=60;encrypt=true;TrustServerCertificate=false",
		},
		{
			"Service Principal Different Port",
			&args.ArgumentList{
				ClientID:     "abcdef12-3456-7890-abcd-ef1234567890",
				TenantID:     "fedcba09-8765-4321-fedc-ba0987654321",
				ClientSecret: "super-secret-key",
				Hostname:     "myserver.example.com",
				Port:         "1444",
				Timeout:      "45",
				EnableSSL:    false,
			},
			"analytics-db",
			"server=myserver.example.com;port=1444;database=analytics-db;user id=abcdef12-3456-7890-abcd-ef1234567890@fedcba09-8765-4321-fedc-ba0987654321;password=super-secret-key;fedauth=ActiveDirectoryServicePrincipal;dial timeout=45;connection timeout=45",
		},
	}

	for _, tc := range testCases {
		if out := CreateAzureADConnectionURL(tc.arg, tc.dbName); out != tc.want {
			t.Errorf("Test Case %s Failed: Expected '%s' got '%s'", tc.name, tc.want, out)
		}
	}
}

func Test_determineAuthMethod(t *testing.T) {
	testCases := []struct {
		name        string
		args        *args.ArgumentList
		expectError bool
		expectType  string
	}{
		{
			"Valid Azure AD Service Principal - All fields provided",
			&args.ArgumentList{
				ClientID:     "12345678-1234-1234-1234-123456789012",
				TenantID:     "87654321-4321-4321-4321-210987654321",
				ClientSecret: "client-secret",
			},
			false,
			"AzureADAuthConnector",
		},
		{
			"Azure AD incomplete - Only ClientID provided",
			&args.ArgumentList{
				ClientID: "12345678-1234-1234-1234-123456789012",
			},
			true,
			"",
		},
		{
			"Azure AD incomplete - Only TenantID provided",
			&args.ArgumentList{
				TenantID: "87654321-4321-4321-4321-210987654321",
			},
			true,
			"",
		},
		{
			"Azure AD incomplete - Only ClientSecret provided",
			&args.ArgumentList{
				ClientSecret: "client-secret",
			},
			true,
			"",
		},
		{
			"Azure AD incomplete - ClientID and TenantID only",
			&args.ArgumentList{
				ClientID: "12345678-1234-1234-1234-123456789012",
				TenantID: "87654321-4321-4321-4321-210987654321",
			},
			true,
			"",
		},
		{
			"Azure AD incomplete - ClientID and ClientSecret only",
			&args.ArgumentList{
				ClientID:     "12345678-1234-1234-1234-123456789012",
				ClientSecret: "client-secret",
			},
			true,
			"",
		},
		{
			"Azure AD incomplete - TenantID and ClientSecret only",
			&args.ArgumentList{
				TenantID:     "87654321-4321-4321-4321-210987654321",
				ClientSecret: "client-secret",
			},
			true,
			"",
		},
		{
			"SQL authentication with Username and Password",
			&args.ArgumentList{
				Username: "sqluser",
				Password: "sqlpassword",
			},
			false,
			"SQLAuthConnector",
		},
		{
			"SQL authentication with only Username",
			&args.ArgumentList{
				Username: "sqluser",
			},
			false,
			"SQLAuthConnector",
		},
		{
			"SQL authentication with only Password",
			&args.ArgumentList{
				Password: "sqlpassword",
			},
			false,
			"SQLAuthConnector",
		},
		{
			"No credentials provided - defaults to SQL",
			&args.ArgumentList{},
			false,
			"SQLAuthConnector",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connector, err := determineAuthMethod(tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("Test case '%s' failed: expected error but got none", tc.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Test case '%s' failed: unexpected error: %v", tc.name, err)
				return
			}

			connectorType := fmt.Sprintf("%T", connector)
			expectedType := fmt.Sprintf("connection.%s", tc.expectType)

			if connectorType != expectedType {
				t.Errorf("Test case '%s' failed: expected connector type %s, got %s", tc.name, expectedType, connectorType)
			}
		})
	}
}
