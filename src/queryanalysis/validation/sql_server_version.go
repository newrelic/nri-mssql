package validation

import (
	"regexp"

	"github.com/blang/semver/v4"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
)

const (
	versionRegexPattern      = `\b(\d+\.\d+\.\d+)\b`
	getSQLServerVersionQuery = "SELECT @@VERSION"
)

var versionRegex = regexp.MustCompile(versionRegexPattern)

func checkSQLServerVersion(sqlConnection *connection.SQLConnection) bool {
	rows, err := sqlConnection.Queryx(getSQLServerVersionQuery)
	if err != nil {
		log.Error("Error getting Server version:", err)
		return false
	}
	defer rows.Close()
	rows.Next()
	var serverVersion string
	if err := rows.Scan(&serverVersion); err != nil {
		log.Error("Error scanning server version:", err)
		return false
	}
	if serverVersion == "" {
		log.Error("Server version is empty")
		return false
	}
	log.Debug("Server version: %s", serverVersion)
	versionStr := versionRegex.FindString(serverVersion)
	if versionStr == "" {
		log.Error("Could not parse version from server version string")
		return false
	}
	log.Debug("Parsed version string: %s", versionStr)
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		log.Error("Error parsing version:", err)
		return false
	}
	log.Debug("Parsed semantic version: %s", version)
	supportedVersions := []uint64{16, 15, 14} // Corresponding to SQL Server 2022, 2019, and 2017
	isSupported := false
	for _, supportedVersion := range supportedVersions {
		if version.Major == supportedVersion {
			isSupported = true
			break
		}
	}
	if !isSupported {
		log.Error("Unsupported SQL Server version: %s", version.String())
	}
	return isSupported
}
