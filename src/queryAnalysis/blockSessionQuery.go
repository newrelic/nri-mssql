package queryAnalysis

import (
	"strconv"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

// blockingSessionQueryDetails struct to hold query results.
type blockingSessionQueryDetails struct {
	BlockingSPID      *int     `db:"blocking_spid"`
	BlockingStatus    *string  `db:"blocking_status"`
	BlockedSPID       *int     `db:"blocked_spid"`
	BlockedStatus     *string  `db:"blocked_status"`
	WaitType          *string  `db:"wait_type"`
	WaitTimeInSeconds *float64 `db:"wait_time_in_seconds"`
	CommandType       *string  `db:"command_type"`
	DatabaseName      *string  `db:"database_name"`
	BlockingQueryText *string  `db:"blocking_query_text"`
	BlockedQueryText  *string  `db:"blocked_query_text"`
}

// AnalyzeBlockingSessions analyzes the SQL Server blocking sessions
func AnalyzeBlockingSessions(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	log.Info("Querying SQL Server for blocking sessions")

	var getBlockingSessionQueryDetailsQuery = `WITH blocking_info AS (
		SELECT
			req.blocking_session_id AS blocking_spid,
			req.session_id AS blocked_spid,
			req.wait_type AS wait_type,
			req.wait_time / 1000.0 AS wait_time_in_seconds,
			req.start_time AS start_time,
			sess.status AS status,
			req.command AS command_type,
			req.database_id AS database_id,
			req.sql_handle AS blocked_sql_handle,
			blocking_req.sql_handle AS blocking_sql_handle,
			blocking_req.start_time AS blocking_start_time
		FROM
			sys.dm_exec_requests AS req
		LEFT JOIN sys.dm_exec_requests AS blocking_req ON blocking_req.session_id = req.blocking_session_id
		LEFT JOIN sys.dm_exec_sessions AS sess ON sess.session_id = req.session_id
		WHERE
			req.blocking_session_id != 0
	)
	SELECT
		blocking_info.blocking_spid,
		blocking_sessions.status AS blocking_status,
		blocking_info.blocked_spid,
		blocked_sessions.status AS blocked_status,
		blocking_info.wait_type,
		blocking_info.wait_time_in_seconds,
		blocking_info.command_type,
		DB_NAME(blocking_info.database_id) AS database_name,
		CASE WHEN blocking_sql.text IS NULL THEN input_buffer.event_info ELSE blocking_sql.text END AS blocking_query_text,
		blocked_sql.text AS blocked_query_text
	FROM
		blocking_info
	JOIN sys.dm_exec_sessions AS blocking_sessions ON blocking_sessions.session_id = blocking_info.blocking_spid
	JOIN sys.dm_exec_sessions AS blocked_sessions ON blocked_sessions.session_id = blocking_info.blocked_spid
	OUTER APPLY sys.dm_exec_sql_text(blocking_info.blocking_sql_handle) AS blocking_sql
	OUTER APPLY sys.dm_exec_sql_text(blocking_info.blocked_sql_handle) AS blocked_sql
	OUTER APPLY sys.dm_exec_input_buffer(blocking_info.blocking_spid, NULL) AS input_buffer
	ORDER BY
		blocking_info.blocking_spid,
		blocking_info.blocked_spid;`

	log.Info("Executing query to get blocking session details.")

	// Slice to hold query results.
	blockingSessionModels := make([]blockingSessionQueryDetails, 0)

	// Execute the query and store the results in the blockingSessionModels slice.
	rows, err := connection.Queryx(getBlockingSessionQueryDetailsQuery)
	if err != nil {
		log.Error("Could not execute query: %s", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var model blockingSessionQueryDetails
		if err := rows.StructScan(&model); err != nil {
			log.Error("Could not scan row: %s", err.Error())
			continue
		}
		blockingSessionModels = append(blockingSessionModels, model)
	}

	log.Info("Number of records retrieved: %d", len(blockingSessionModels))

	// Log and report each result from the query.
	for _, model := range blockingSessionModels {
		if model.BlockingSPID == nil || model.BlockedSPID == nil {
			log.Warn("Skipping entry with nil field: BlockingSPID or BlockedSPID")
			continue // Skip this entry if any critical field is nil
		}

		blockingSPID := *model.BlockingSPID
		blockingStatus := ""
		if model.BlockingStatus != nil {
			blockingStatus = *model.BlockingStatus
		}
		blockedSPID := *model.BlockedSPID
		blockedStatus := ""
		if model.BlockedStatus != nil {
			blockedStatus = *model.BlockedStatus
		}
		waitType := ""
		if model.WaitType != nil {
			waitType = *model.WaitType
		}
		waitTimeInSeconds := 0.0
		if model.WaitTimeInSeconds != nil {
			waitTimeInSeconds = *model.WaitTimeInSeconds
		}
		commandType := ""
		if model.CommandType != nil {
			commandType = *model.CommandType
		}
		databaseName := ""
		if model.DatabaseName != nil {
			databaseName = *model.DatabaseName
		}
		blockingQueryText := ""
		if model.BlockingQueryText != nil {
			blockingQueryText = *model.BlockingQueryText
		}
		blockedQueryText := ""
		if model.BlockedQueryText != nil {
			blockedQueryText = *model.BlockedQueryText
		}

		metricSet := instanceEntity.NewMetricSet("MssqlBlockingSessions",
			attribute.Attribute{Key: "blockingSPID", Value: strconv.Itoa(blockingSPID)},
			attribute.Attribute{Key: "blockedSPID", Value: strconv.Itoa(blockedSPID)},
		)

		// Add all the fields to the metric set.
		if model.BlockingStatus != nil {
			metricSet.SetMetric("blockingStatus", blockingStatus, metric.ATTRIBUTE)
		}
		if model.BlockedStatus != nil {
			metricSet.SetMetric("blockedStatus", blockedStatus, metric.ATTRIBUTE)
		}
		if model.WaitType != nil {
			metricSet.SetMetric("waitType", waitType, metric.GAUGE)
		}
		if model.WaitTimeInSeconds != nil {
			metricSet.SetMetric("waitTimeInSeconds", waitTimeInSeconds, metric.GAUGE)
		}
		if model.CommandType != nil {
			metricSet.SetMetric("commandType", commandType, metric.ATTRIBUTE)
		}
		if model.DatabaseName != nil {
			metricSet.SetMetric("databaseName", databaseName, metric.ATTRIBUTE)
		}
		if model.BlockingQueryText != nil {
			metricSet.SetMetric("blockingQueryText", blockingQueryText, metric.ATTRIBUTE)
		}
		if model.BlockedQueryText != nil {
			metricSet.SetMetric("blockedQueryText", blockedQueryText, metric.ATTRIBUTE)
		}
	}

	log.Info("Completed processing all blocking session entries.")
}
