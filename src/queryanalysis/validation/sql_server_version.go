package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/newrelic/nri-mssql/src/connection"

	"github.com/blang/semver/v4"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
)

const (
	versionRegexPattern          = `\b(\d+\.\d+\.\d+)\b`
	getSQLServerVersionQuery     = "SELECT @@VERSION"
	lastSupportedVersion         = 16
	firstSupportedVersion        = 14
	dmvOnlyFirstSupportedVersion = 13 // SQL Server 2016 (AT TIME ZONE, STRING_SPLIT)
	// Defines the supported version range for Azure SQL Server in the cloud, from version 12 to 16.
	azureFirstSupportedVersion = 12
	azureLastSupportedVersion  = 16
)

var (
	versionRegex = regexp.MustCompile(versionRegexPattern)
)

func getSQLServerVersion(sqlConnection *connection.SQLConnection) (string, error) {
	rows, err := sqlConnection.Queryx(getSQLServerVersionQuery)
	if err != nil {
		return "", fmt.Errorf("error getting server version: %w", err)
	}
	defer rows.Close()
	rows.Next()
	var serverVersion string
	if err := rows.Scan(&serverVersion); err != nil {
		return "", fmt.Errorf("error scanning server version: %w", err)
	}
	log.Debug("Server version: %s", serverVersion)
	return serverVersion, nil
}

func parseSQLServerVersion(serverVersion string) (semver.Version, error) {
	versionStr := versionRegex.FindString(serverVersion)
	log.Debug("Parsed version string: %s", versionStr)
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		return semver.Version{}, fmt.Errorf("error parsing version: %w", err)
	}
	log.Debug("Parsed semantic version: %s", version)
	return version, nil
}

func checkSQLServerVersion(sqlConnection *connection.SQLConnection, isDMVOnlyMode bool) (bool, error) {
	serverVersion, err := getSQLServerVersion(sqlConnection)
	if err != nil {
		return false, err
	}
	version, err := parseSQLServerVersion(serverVersion)
	if err != nil {
		return false, err
	}

	isAzure := strings.Contains(strings.ToLower(serverVersion), "azure")

	if isAzure {
		// Azure: Keep existing range (12-16) for both modes
		return version.Major >= azureFirstSupportedVersion &&
			version.Major <= azureLastSupportedVersion, nil
	}

	// On-premises SQL Server
	if isDMVOnlyMode {
		// DMV-only mode: Support SQL Server 2016+ (version 13+)
		return version.Major >= dmvOnlyFirstSupportedVersion &&
			version.Major <= lastSupportedVersion, nil
	}

	// Query Store mode: Require SQL Server 2017+ (version 14+)
	return version.Major >= firstSupportedVersion &&
		version.Major <= lastSupportedVersion, nil
}
