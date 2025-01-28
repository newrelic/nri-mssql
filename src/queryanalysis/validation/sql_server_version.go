package validation

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/blang/semver/v4"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
)

const (
	versionRegexPattern      = `\b(\d+\.\d+\.\d+)\b`
	getSQLServerVersionQuery = "SELECT @@VERSION"
)

var (
	versionRegex          = regexp.MustCompile(versionRegexPattern)
	errEmptyServerVersion = errors.New("server version is empty")
	errParseVersion       = errors.New("could not parse version from server version string")
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
	if serverVersion == "" {
		return "", fmt.Errorf("%w", errEmptyServerVersion)
	}
	log.Debug("Server version: %s", serverVersion)
	return serverVersion, nil
}

func parseSQLServerVersion(serverVersion string) (semver.Version, error) {
	versionStr := versionRegex.FindString(serverVersion)
	if versionStr == "" {
		return semver.Version{}, fmt.Errorf("%w", errParseVersion)
	}
	log.Debug("Parsed version string: %s", versionStr)
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		return semver.Version{}, fmt.Errorf("error parsing version: %w", err)
	}
	log.Debug("Parsed semantic version: %s", version)
	return version, nil
}

func isSQLServerVersionSupported(version semver.Version) bool {
	supportedVersions := []uint64{16, 15, 14} // Corresponding to SQL Server 2022, 2019, and 2017
	for _, supportedVersion := range supportedVersions {
		if version.Major == supportedVersion {
			return true
		}
	}
	log.Error("Unsupported SQL Server version: %s", version.String())
	return false
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
	return isSQLServerVersionSupported(version), nil
}
