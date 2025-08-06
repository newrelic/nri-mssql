module github.com/newrelic/nri-mssql

go 1.24.6

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/microsoft/go-mssqldb v1.9.2
	github.com/newrelic/infra-integrations-sdk/v3 v3.9.1
	github.com/stretchr/testify v1.10.0
	github.com/xeipuuv/gojsonschema v1.2.0
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Allows TLS certs with negative serial numbers.
// Before go 1.23 these certificates where accepted, now the corresponding go debug variable is needed
// to restore the previous behavior
// <https://cs.opensource.google/go/go/+/refs/tags/go1.23.1:src/crypto/x509/parser.go;l=1019>
godebug x509negativeserial=1
