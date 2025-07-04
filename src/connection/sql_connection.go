// Package connection contains the SQLConnection type and methods for manipulating and querying the connection
package connection

import (
	"fmt"
	"net/url"
	"strconv"

	// go-mssqldb is required for mssql driver but isn't used in code
	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/azuread"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
)

// SQLConnection represents a wrapper around a SQL Server connection
type SQLConnection struct {
	Connection *sqlx.DB
	Host       string
}

type AuthConnector interface {
	Connect(args *args.ArgumentList, dbName string) (*sqlx.DB, error)
}

type SQLAuthConnector struct{}

func (s SQLAuthConnector) Connect(args *args.ArgumentList, dbName string) (*sqlx.DB, error) {
	connectionURL := CreateConnectionURL(args, dbName)
	return sqlx.Connect("mssql", connectionURL)
}

type AzureADAuthConnector struct{}

func (a AzureADAuthConnector) Connect(args *args.ArgumentList, dbName string) (*sqlx.DB, error) {
	connectionURL := CreateAzureADConnectionURL(args, dbName)
	return sqlx.Connect(azuread.DriverName, connectionURL)
}

func isAzureADServicePrincipalAuth(args *args.ArgumentList) bool {
	return args.ClientID != "" && args.TenantID != "" && args.ClientSecret != ""
}

func determineAuthMethod(args *args.ArgumentList) (AuthConnector, error) {
	switch {
	case isAzureADServicePrincipalAuth(args):
		log.Debug("Detected Azure AD Service Principal authentication - using ClientID, TenantID, and ClientSecret")
		return AzureADAuthConnector{}, nil
	default:
		// Check for incomplete Azure AD credentials first
		azureFieldsProvided := 0
		if args.ClientID != "" {
			azureFieldsProvided++
		}
		if args.TenantID != "" {
			azureFieldsProvided++
		}
		if args.ClientSecret != "" {
			azureFieldsProvided++
		}

		if azureFieldsProvided > 0 && azureFieldsProvided < 3 {
			return nil, fmt.Errorf("incomplete Azure AD Service Principal credentials: all three fields (ClientID, TenantID, ClientSecret) must be provided together")
		}

		// Default to SQL authentication (supports Windows Auth, SQL Auth with credentials, etc.)
		log.Debug("Using SQL Server authentication")
		return SQLAuthConnector{}, nil
	}
}

func createConnectionWithAuth(args *args.ArgumentList, dbName string) (*SQLConnection, error) {
	connector, err := determineAuthMethod(args)
	if err != nil {
		return nil, fmt.Errorf("failed to determine authentication method: %w", err)
	}

	db, err := connector.Connect(args, dbName)
	if err != nil {
		return nil, err
	}
	return &SQLConnection{
		Connection: db,
		Host:       args.Hostname,
	}, nil
}

func NewConnection(args *args.ArgumentList) (*SQLConnection, error) {
	return createConnectionWithAuth(args, "")
}

// package-level variable to hold the original function which is needed to mock this NewDatabaseConnection for unit testing.
var CreateDatabaseConnection = NewDatabaseConnection

func NewDatabaseConnection(args *args.ArgumentList, dbName string) (*SQLConnection, error) {
	return createConnectionWithAuth(args, dbName)
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
	log.Debug("Running query: %s", query)
	return sc.Connection.Select(v, query)
}

// Queryx runs a query and returns a set of rows
func (sc SQLConnection) Queryx(query string) (*sqlx.Rows, error) {
	return sc.Connection.Queryx(query)
}

// CreateConnectionURL tags in args and creates the connection string.
// All args should be validated before calling this.
func CreateConnectionURL(args *args.ArgumentList, dbName string) string {
	connectionString := ""

	connectionURL := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(args.Username, args.Password),
		Host:   args.Hostname,
	}

	// If port is present use port if not user instance
	if args.Port != "" {
		connectionURL.Host = fmt.Sprintf("%s:%s", connectionURL.Host, args.Port)
	} else {
		connectionURL.Path = args.Instance
	}

	// Format query parameters
	query := url.Values{}
	query.Add("dial timeout", args.Timeout)
	query.Add("connection timeout", args.Timeout)
	if dbName != "" {
		query.Add("database", dbName)
		log.Debug("using database name : %s as a query parameter in the connection url", dbName)
	}

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

	return connectionString
}

// CreateAzureADConnectionURL creates a connection string specifically for Azure AD authentication.
func CreateAzureADConnectionURL(args *args.ArgumentList, dbName string) string {
	connectionString := fmt.Sprintf(
		"server=%s;port=%s;database=%s;user id=%s@%s;password=%s;fedauth=ActiveDirectoryServicePrincipal;dial timeout=%s;connection timeout=%s",
		args.Hostname,
		args.Port,
		dbName,
		args.ClientID,     // Client ID
		args.TenantID,     // Tenant ID
		args.ClientSecret, // Client Secret
		args.Timeout,
		args.Timeout,
	)

	if args.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(args.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				connectionString += fmt.Sprintf(";%s=%s", k, v[0])
			}
		} else {
			log.Warn("Could not successfully parse ExtraConnectionURLArgs.", err.Error())
		}
	}

	if args.EnableSSL {
		connectionString += ";encrypt=true"
		if args.TrustServerCertificate {
			connectionString += ";TrustServerCertificate=true"
		} else {
			connectionString += ";TrustServerCertificate=false"
			if args.CertificateLocation != "" {
				connectionString += fmt.Sprintf(";certificate=%s", args.CertificateLocation)
			}
		}
	}

	return connectionString
}
