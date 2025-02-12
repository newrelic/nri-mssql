package validation

import (
	"fmt"
	"regexp"

	"github.com/newrelic/nri-mssql/src/connection"

	"github.com/blang/semver/v4"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
)

const (
	versionRegexPattern      = `\b(\d+\.\d+\.\d+)\b`
	getSQLServerVersionQuery = "SELECT @@VERSION"
	lastSupportedVersion     = 16
	firstSupportedVersion    = 14
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

func checkSQLServerVersion(sqlConnection *connection.SQLConnection) (bool, error) {
	serverVersion, err := getSQLServerVersion(sqlConnection)
	if err != nil {
		return false, err
	}
	version, err := parseSQLServerVersion(serverVersion)
	if err != nil {
		return false, err
	}
	return version.Major >= firstSupportedVersion && version.Major <= lastSupportedVersion, nil
}
