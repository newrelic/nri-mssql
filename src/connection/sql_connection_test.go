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

func Test_isAzureADServicePrincipal(t *testing.T) {
	testCases := []struct {
		name     string
		args     *args.ArgumentList
		expected bool
	}{
		{
			"Valid Azure AD Service Principal format",
			&args.ArgumentList{
				Username: "12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321",
				Password: "client-secret",
			},
			true,
		},
		{
			"Valid Azure AD Service Principal with complex password",
			&args.ArgumentList{
				Username: "abcdef12-3456-7890-abcd-ef1234567890@fedcba09-8765-4321-fedc-ba0987654321",
				Password: "MyVerySecretClientSecret123!@#",
			},
			true,
		},
		{
			"Regular SQL username and password",
			&args.ArgumentList{
				Username: "sqluser",
				Password: "sqlpassword",
			},
			false,
		},
		{
			"Email-like username (not Azure AD)",
			&args.ArgumentList{
				Username: "user@company.com",
				Password: "password123",
			},
			false,
		},
		{
			"Username with @ but not UUID format",
			&args.ArgumentList{
				Username: "myapp@domain",
				Password: "secret",
			},
			false,
		},
		{
			"Malformed UUID - missing hyphens",
			&args.ArgumentList{
				Username: "123456781234123412341234567890ab@876543214321432143212109876543cd",
				Password: "secret",
			},
			false,
		},
		{
			"UUID format but wrong length",
			&args.ArgumentList{
				Username: "1234-1234-1234-1234@5678-5678-5678-5678",
				Password: "secret",
			},
			false,
		},
		{
			"Valid UUID format but hyphens in wrong positions",
			&args.ArgumentList{
				Username: "12345678-123-41234-1234-123456789012@87654321-432-14321-4321-210987654321",
				Password: "secret",
			},
			false,
		},
		{
			"Empty username",
			&args.ArgumentList{
				Username: "",
				Password: "secret",
			},
			false,
		},
		{
			"Empty password",
			&args.ArgumentList{
				Username: "12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321-210987654321",
				Password: "",
			},
			false,
		},
		{
			"Both username and password empty",
			&args.ArgumentList{
				Username: "",
				Password: "",
			},
			false,
		},
		{
			"Username too short",
			&args.ArgumentList{
				Username: "short@string",
				Password: "secret",
			},
			false,
		},
		{
			"Multiple @ symbols",
			&args.ArgumentList{
				Username: "12345678-1234-1234-1234-123456789012@87654321-4321-4321-4321@210987654321",
				Password: "secret",
			},
			false,
		},
		{
			"No @ symbol",
			&args.ArgumentList{
				Username: "12345678-1234-1234-1234-12345678901287654321-4321-4321-4321-210987654321",
				Password: "secret",
			},
			false,
		},
		{
			"UUID with uppercase letters",
			&args.ArgumentList{
				Username: "ABCDEF12-3456-7890-ABCD-EF1234567890@FEDCBA09-8765-4321-FEDC-BA0987654321",
				Password: "secret",
			},
			true,
		},
		{
			"Mixed case UUID",
			&args.ArgumentList{
				Username: "AbCdEf12-3456-7890-AbCd-Ef1234567890@FeDcBa09-8765-4321-FeDc-Ba0987654321",
				Password: "secret",
			},
			true,
		},
		{
			"Valid format with numbers only",
			&args.ArgumentList{
				Username: "12345678-1234-5678-9012-345678901234@98765432-1098-7654-3210-987654321098",
				Password: "secret",
			},
			true,
		},
		{
			"Valid format with letters only",
			&args.ArgumentList{
				Username: "abcdefgh-ijkl-mnop-qrst-uvwxyzabcdef@ghijklmn-opqr-stuv-wxyz-abcdefghijkl",
				Password: "secret",
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAzureADServicePrincipal(tc.args)
			if result != tc.expected {
				t.Errorf("Test case '%s' failed: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}
