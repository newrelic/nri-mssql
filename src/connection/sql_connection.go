// Package connection contains the SQLConnection type and methods for manipulating and querying the connection
package connection

import (
	"fmt"
	"net/url"
	"strconv"

	// go-mssqldb is required for mssql driver but isn't used in code
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/args"
)

// SQLConnection represents a wrapper around a SQL Server connection
type SQLConnection struct {
	Connection *sqlx.DB
	Host       string
}

// NewConnection creates a new SQLConnection from args
func NewConnection(args *args.ArgumentList) (*SQLConnection, error) {
	db, err := sqlx.Connect("mssql", CreateConnectionURL(args))
	if err != nil {
		return nil, err
	}
	return &SQLConnection{
		Connection: db,
		Host:       args.Hostname,
	}, nil
}

// Close closes the SQL connection. If an error occurs
// it is logged as a warning.
func (sc SQLConnection) Close() {
	if err := sc.Connection.Close(); err != nil {
		log.Warn("Unable to close SQL Connection: %s", err.Error())
	}
}

// Query runs a query and loads results into v
func (sc SQLConnection) Query(v interface{}, query string) error {
	return sc.Connection.Select(v, query)
}

// Queryx runs a query and returns a set of rows
func (sc SQLConnection) Queryx(query string) (*sqlx.Rows, error) {
	return sc.Connection.Queryx(query)
}

// CreateConnectionURL tags in args and creates the connection string.
// All args should be validated before calling this.
func CreateConnectionURL(args *args.ArgumentList) string {
	connectionString := ""
	connectionURL := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(args.Username, args.Password),
		Host:   args.Hostname,
	}

	if args.Instance != "" {
		connectionURL.Path = args.Instance
	} else {
		connectionURL.Host = fmt.Sprintf("%s:%s", connectionURL.Host, args.Port)
	}

	// Format query parameters
	query := url.Values{}
	query.Add("database", args.Database)
	query.Add("dial timeout", args.Timeout)
	query.Add("connection timeout", args.Timeout)

	if args.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(args.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				query.Add(k, v[0])
			}
		} else {
			log.Warn("Could not successfully parse ExtraConnectionURLArgs.", err.Error())
		}
	}

	if args.EnableSSL {
		query.Add("encrypt", "true")
		query.Add("TrustServerCertificate", strconv.FormatBool(args.TrustServerCertificate))
		if !args.TrustServerCertificate {
			query.Add("certificate", args.CertificateLocation)
		}
	}

	connectionURL.RawQuery = query.Encode()
	connectionString = connectionURL.String()
	log.Debug("CreateConnectionURL: url: %s", connectionString)
	return connectionString
}
