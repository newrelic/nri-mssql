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
		name string
		arg  *args.ArgumentList
		want string
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
			"sqlserver://user:pass@localhost:1443?applicationIntent=true&connection+timeout=30&dial+timeout=30",
		},
	}

	for _, tc := range testCases {
		if out := CreateConnectionURL(tc.arg); out != tc.want {
			t.Errorf("Test Case %s Failed: Expected '%s' got '%s'", tc.name, tc.want, out)
		}
	}
}
