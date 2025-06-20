package common

import (
	"strings"

	"github.com/newrelic/nri-mssql/src/database"
)

var UnsupportedQueryPatterns = map[int][]string{
	database.AzureSQLDatabaseEngineEditionNumber: { // Azure SQL Database EngineEdition
		"sys.dm_os_process_memory",
		"sys.master_files",
		"exec sp_configure",
		"sys.dm_os_sys_memory",
		"sys.dm_os_volume_stats",
	},
}

func SkipQueryForEngineEdition(engineEdition int, query string) bool {
	for _, pattern := range UnsupportedQueryPatterns[engineEdition] {
		if strings.Contains(strings.ToLower(query), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
