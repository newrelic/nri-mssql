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
				Username:  "client-id@tenant-id",
				Password:  "client-secret",
				Hostname:  "localhost",
				Port:      "1433",
				Timeout:   "30",
				EnableSSL: false,
			},
			"",
			"server=localhost;port=1433;database=;user id=client-id@tenant-id;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30",
		},
		{
			"Service Principal with Database",
			&args.ArgumentList{
				Username:  "client-id@tenant-id",
				Password:  "client-secret",
				Hostname:  "sqlserver.database.windows.net",
				Port:      "1433",
				Timeout:   "30",
				EnableSSL: false,
			},
			"test-database",
			"server=sqlserver.database.windows.net;port=1433;database=test-database;user id=client-id@tenant-id;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30",
		},
		{
			"Service Principal SSL Trust Certificate",
			&args.ArgumentList{
				Username:               "client-id@tenant-id",
				Password:               "client-secret",
				Hostname:               "sqlserver.database.windows.net",
				Port:                   "1433",
				Timeout:                "30",
				EnableSSL:              true,
				TrustServerCertificate: true,
			},
			"production-db",
			"server=sqlserver.database.windows.net;port=1433;database=production-db;user id=client-id@tenant-id;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30;encrypt=true;TrustServerCertificate=true",
		},
		{
			"Service Principal SSL with Certificate File",
			&args.ArgumentList{
				Username:               "client-id@tenant-id",
				Password:               "client-secret",
				Hostname:               "sqlserver.database.windows.net",
				Port:                   "1433",
				Timeout:                "30",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "/path/to/cert.pem",
			},
			"secure-db",
			"server=sqlserver.database.windows.net;port=1433;database=secure-db;user id=client-id@tenant-id;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=30;connection timeout=30;encrypt=true;TrustServerCertificate=false;certificate=/path/to/cert.pem",
		},
		{
			"Service Principal SSL Don't Trust No Certificate",
			&args.ArgumentList{
				Username:               "client-id@tenant-id",
				Password:               "client-secret",
				Hostname:               "localhost",
				Port:                   "1433",
				Timeout:                "60",
				EnableSSL:              true,
				TrustServerCertificate: false,
				CertificateLocation:    "",
			},
			"test-db",
			"server=localhost;port=1433;database=test-db;user id=client-id@tenant-id;password=client-secret;fedauth=ActiveDirectoryServicePrincipal;dial timeout=60;connection timeout=60;encrypt=true;TrustServerCertificate=false",
		},
		{
			"Service Principal Different Port",
			&args.ArgumentList{
				Username:  "app-registration@azure-tenant",
				Password:  "super-secret-key",
				Hostname:  "myserver.example.com",
				Port:      "1444",
				Timeout:   "45",
				EnableSSL: false,
			},
			"analytics-db",
			"server=myserver.example.com;port=1444;database=analytics-db;user id=app-registration@azure-tenant;password=super-secret-key;fedauth=ActiveDirectoryServicePrincipal;dial timeout=45;connection timeout=45",
		},
	}

	for _, tc := range testCases {
		if out := CreateAzureADConnectionURL(tc.arg, tc.dbName); out != tc.want {
			t.Errorf("Test Case %s Failed: Expected '%s' got '%s'", tc.name, tc.want, out)
		}
	}
}
