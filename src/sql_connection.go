package main

import (
	"fmt"
	"net/url"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// SQLConnection represents a wrapper around a SQL Server connection
type SQLConnection struct {
	connection *sqlx.DB
}

// newConnection creates a new sqlConnection from args
func newConnection() (*SQLConnection, error) {
	db, err := sqlx.Connect("mssql", createConnectionURL())
	if err != nil {
		return nil, err
	}
	return &SQLConnection{
		connection: db,
	}, nil
}

// Close closes the SQL connection. If an error occurs
// it is logged as a warning.
func (sc SQLConnection) Close() {
	if err := sc.connection.Close(); err != nil {
		log.Warn("Unable to close SQL Connection: %s", err.Error())
	}
}

// Query runs a query and loads results into v
func (sc SQLConnection) Query(v interface{}, query string) error {
	return sc.connection.Select(v, query)
}

// createConnectionURL tags in args and creates the connection string.
// All args should be validated before calling this.
func createConnectionURL() string {
	connectionURL := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(args.Username, args.Password),
		Host:   args.Hostname,
	}

	// If port is present use port if not user instace
	if args.Port != "" {
		connectionURL.Host = fmt.Sprintf("%s:%s", connectionURL.Host, args.Port)
	} else {
		connectionURL.Path = args.Instance
	}

	// Format query parameters
	query := url.Values{}
	query.Add("dial timeout", args.Timeout)

	if args.EnableSSL {
		query.Add("encrypt", "true")

		query.Add("TrustServerCertificate", strconv.FormatBool(args.TrustServerCertificate))

		if !args.TrustServerCertificate {
			query.Add("certificate", args.CertificateLocation)
		}
	}

	connectionURL.RawQuery = query.Encode()

	connectionString := connectionURL.String()
	return connectionString
}
