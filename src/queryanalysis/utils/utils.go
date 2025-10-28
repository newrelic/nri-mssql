package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/instance"
	"github.com/newrelic/nri-mssql/src/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"

	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/config"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

var (
	ErrUnknownQueryType       = errors.New("unknown query type")
	ErrCreatingInstanceEntity = errors.New("error creating instance entity")
	// literalAnonymizer is a regular expression pattern used to match and identify
	// certain types of literal values in a string. Specifically, it matches:
	// 1. Single-quoted character sequences, such as 'example'.
	// 2. Numeric sequences (integer numbers), such as 123 or 456.
	// 3. Double-quoted strings, such as "example".
	// This regex can be useful for identifying and potentially anonymizing literal values
	// in a given text, like extracting or concealing specific data within strings.
	literalAnonymizer = regexp.MustCompile(`'[^']*'|\d+|".*?"`)
)

// queryFormatter defines a function type for formatting a query string.
type queryFormatter func(query string, args args.ArgumentList) string

// queryFormatters maps query types to their corresponding formatting functions.
var queryFormatters = map[string]queryFormatter{
	"slowQueries":      formatSlowQueries,
	"waitAnalysis":     formatWaitAnalysis,
	"blockingSessions": formatBlockingSessions,
}

// formatSlowQueries formats the slow queries query.
func formatSlowQueries(query string, args args.ArgumentList) string {
	return fmt.Sprintf(query, args.QueryMonitoringFetchInterval, config.TextTruncateLimit)
}

// formatWaitAnalysis formats the wait analysis query.
func formatWaitAnalysis(query string, args args.ArgumentList) string {
	return fmt.Sprintf(query, args.QueryMonitoringCountThreshold, config.TextTruncateLimit)
}

// formatBlockingSessions formats the blocking sessions query.
func formatBlockingSessions(query string, args args.ArgumentList) string {
	return fmt.Sprintf(query, args.QueryMonitoringCountThreshold, config.TextTruncateLimit)
}

// LoadQueries loads and formats query details based on the provided arguments.
func LoadQueries(queries []models.QueryDetailsDto, arguments args.ArgumentList) ([]models.QueryDetailsDto, error) {
	loadedQueries := make([]models.QueryDetailsDto, len(queries))
	copy(loadedQueries, queries) // Create a copy to avoid modifying the original

	for i := range loadedQueries {
		formatter, ok := queryFormatters[loadedQueries[i].Type]
		if !ok {
			// Log the error and return an error instead of nil
			err := fmt.Errorf("%w: %s", ErrUnknownQueryType, loadedQueries[i].Type)
			return nil, err
		}
		loadedQueries[i].Query = formatter(loadedQueries[i].Query, arguments)
	}
	return loadedQueries, nil
}

func ExecuteQuery(arguments args.ArgumentList, queryDetailsDto models.QueryDetailsDto, integration *integration.Integration, sqlConnection *connection.SQLConnection) ([]interface{}, error) {
	log.Debug("Executing query: %s", queryDetailsDto.Query)
	rows, err := sqlConnection.Connection.Queryx(queryDetailsDto.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()
	log.Debug("Query executed: %s", queryDetailsDto.Query)
	result, queryIDs, err := BindQueryResults(arguments, rows, queryDetailsDto, integration, sqlConnection)
	rows.Close()

	// Process collected query IDs for execution plan
	if len(queryIDs) > 0 {
		ProcessExecutionPlans(arguments, integration, sqlConnection, queryIDs)
	}
	return result, err
}

// BindQueryResults binds query results to the specified data model using `sqlx`
// nolint:gocyclo
func BindQueryResults(arguments args.ArgumentList,
	rows *sqlx.Rows,
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection) ([]interface{}, []models.HexString, error) {
	results := make([]interface{}, 0)
	queryIDs := make([]models.HexString, 0) // List to collect queryIDs for all slowQueries to process execution plans

	// For slowQueries, collect all enriched queries first, then filter
	var enrichedSlowQueries []EnrichedSlowQueryDetails

	for rows.Next() {
		switch queryDetailsDto.Type {
		case "slowQueries":
			var model models.TopNSlowQueryDetails
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan row: ", err)
				continue
			}
			if model.QueryText != nil {
				*model.QueryText = AnonymizeQueryText(*model.QueryText)
			}
			
			// Create an enriched model with calculated averages
			enrichedModel := EnrichSlowQueryWithAverages(model)
			enrichedSlowQueries = append(enrichedSlowQueries, enrichedModel)

			// Collect query IDs for fetching executionPlans
			if model.QueryID != nil {
				queryIDs = append(queryIDs, *model.QueryID)
			}

		case "waitAnalysis":
			var model models.WaitTimeAnalysis
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan row: ", err)
				continue
			}
			if model.QueryText != nil {
				*model.QueryText = AnonymizeQueryText(*model.QueryText)
			}
			results = append(results, model)
		case "blockingSessions":
			var model models.BlockingSessionQueryDetails
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan row: ", err)
				continue
			}
			if model.BlockingQueryText != nil {
				*model.BlockingQueryText = AnonymizeQueryText(*model.BlockingQueryText)
			}
			if model.BlockedQueryText != nil {
				*model.BlockedQueryText = AnonymizeQueryText(*model.BlockedQueryText)
			}
			results = append(results, model)
		default:
			return nil, queryIDs, fmt.Errorf("%w: %s", ErrUnknownQueryType, queryDetailsDto.Type)
		}
	}

	// Apply filtering logic for slowQueries
	if queryDetailsDto.Type == "slowQueries" && len(enrichedSlowQueries) > 0 {
		// Filter and limit the slow queries based on thresholds
		filteredQueries, filterMetrics := FilterSlowQueriesWithMetrics(enrichedSlowQueries, arguments)
		
		// Log filtering metrics for debugging
		LogFilterMetrics(filterMetrics)
		
		// Convert filtered queries back to []interface{}
		results = make([]interface{}, len(filteredQueries))
		for i, query := range filteredQueries {
			results[i] = query
		}
	}

	return results, queryIDs, nil
}

// ProcessExecutionPlans processes execution plans for all collected queryIDs
func ProcessExecutionPlans(arguments args.ArgumentList, integration *integration.Integration, sqlConnection *connection.SQLConnection, queryIDs []models.HexString) {
	if len(queryIDs) == 0 {
		return
	}
	stringIDs := make([]string, len(queryIDs))
	for i, qid := range queryIDs {
		stringIDs[i] = string(qid) // Cast HexString to string
	}

	// Join the converted string slice into a comma-separated list
	queryIDString := strings.Join(stringIDs, ",")

	GenerateAndIngestExecutionPlan(arguments, integration, sqlConnection, queryIDString)
}

func GenerateAndIngestExecutionPlan(arguments args.ArgumentList, integration *integration.Integration, sqlConnection *connection.SQLConnection, queryIDString string) {
	executionPlanQuery := fmt.Sprintf(config.ExecutionPlanQueryTemplate, min(config.IndividualQueryCountMax, arguments.QueryMonitoringCountThreshold),
		arguments.QueryMonitoringResponseTimeThreshold, queryIDString, arguments.QueryMonitoringFetchInterval, config.TextTruncateLimit)

	var model models.ExecutionPlanResult

	rows, err := sqlConnection.Connection.Queryx(executionPlanQuery)
	if err != nil {
		log.Error("Failed to execute execution plan query: %s", err)
		return
	}
	defer rows.Close()

	results := make([]interface{}, 0)

	for rows.Next() {
		if err := rows.StructScan(&model); err != nil {
			log.Error("Could not scan execution plan row: %s", err)
			return
		}
		*model.SQLText = AnonymizeQueryText(*model.SQLText)
		results = append(results, model)
	}

	queryDetailsDto := models.QueryDetailsDto{
		EventName: "MSSQLQueryExecutionPlans",
	}

	// Ingest the execution plan
	if err := IngestQueryMetricsInBatches(results, queryDetailsDto, integration, sqlConnection); err != nil {
		log.Error("Failed to ingest execution plan: %s", err)
	}
}

func IngestQueryMetricsInBatches(results []interface{},
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection,
) error {
	for start := 0; start < len(results); start += config.BatchSize {
		end := start + config.BatchSize
		if end > len(results) {
			end = len(results)
		}

		batchResult := results[start:end]

		if err := IngestQueryMetrics(batchResult, queryDetailsDto, integration, sqlConnection); err != nil {
			return fmt.Errorf("error ingesting batch from %d to %d: %w", start, end, err)
		}
	}

	return nil
}

func convertResultToMap(result interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling result: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(data, &resultMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling to map: %w", err)
	}
	return resultMap, nil
}

// handleGaugeMetric processes the gauge metric and logs any errors encountered
func handleGaugeMetric(key, strValue string, metricSet *metric.Set) {
	floatValue, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		log.Error("failed to parse float value for key %s: %v", key, err)
		return
	}

	err = metricSet.SetMetric(key, floatValue, metric.GAUGE)
	if err != nil {
		log.Error("failed to set metric for key %s: %v", key, err)
	}
}

// IngestQueryMetrics processes and ingests query metrics into the New Relic entity
func IngestQueryMetrics(results []interface{}, queryDetailsDto models.QueryDetailsDto, integration *integration.Integration, sqlConnection *connection.SQLConnection) error {
	instanceEntity, err := instance.CreateInstanceEntity(integration, sqlConnection)
	if err != nil {
		log.Error("%w: %v", ErrCreatingInstanceEntity, err)
	}

	for _, result := range results {
		var dataToSend interface{}
		
		// For slow queries, convert to NewRelic format (exclude totals)
		if queryDetailsDto.Type == "slowQueries" {
			if enrichedQuery, ok := result.(EnrichedSlowQueryDetails); ok {
				dataToSend = enrichedQuery.ToNewRelicFormat()
			} else {
				dataToSend = result
			}
		} else {
			dataToSend = result
		}

		// Convert the result into a map[string]interface{} for dynamic key-value access
		resultMap, err := convertResultToMap(dataToSend)
		if err != nil {
			log.Error("failed to convert result: %v", err)
			continue
		}

		// Create a new metric set with the query name
		metricSet := instanceEntity.NewMetricSet(queryDetailsDto.EventName)

		// Iterate over the map and add each key-value pair as a metric
		for key, value := range resultMap {
			strValue := fmt.Sprintf("%v", value) // Convert the value to a string representation
			metricType := metrics.DetectMetricType(strValue)
			if metricType == metric.GAUGE {
				handleGaugeMetric(key, strValue, metricSet)
			} else {
				if err := metricSet.SetMetric(key, strValue, metric.ATTRIBUTE); err != nil {
					// Handle the error. This could be logging, returning the error, etc.
					log.Error("failed to set metric: %v", err)
				}
			}
		}
	}
	err = integration.Publish()
	if err != nil {
		return err
	}
	return nil
}

func AnonymizeQueryText(query string) string {
	anonymizedQuery := literalAnonymizer.ReplaceAllString(query, "?")
	return anonymizedQuery
}

// ValidateAndSetDefaults checks if fields are invalid and sets defaults
func ValidateAndSetDefaults(args *args.ArgumentList) {
	// Since EnableQueryMonitoring is a boolean, no need to reset as it can't be invalid in this context
	if args.QueryMonitoringResponseTimeThreshold < 0 {
		args.QueryMonitoringResponseTimeThreshold = config.QueryResponseTimeThresholdDefault
		log.Warn("Query response time threshold is negative, setting to default value: %d", config.QueryResponseTimeThresholdDefault)
	}

	if args.QueryMonitoringCountThreshold < 0 {
		args.QueryMonitoringCountThreshold = config.SlowQueryCountThresholdDefault
		log.Warn("Query count threshold is negative, setting to default value: %d", config.SlowQueryCountThresholdDefault)
	} else if args.QueryMonitoringCountThreshold >= config.GroupedQueryCountMax {
		args.QueryMonitoringCountThreshold = config.GroupedQueryCountMax
		log.Warn("Query count threshold is greater than max supported value, setting to max supported value: %d", config.GroupedQueryCountMax)
	}
}

// CalculateAvgCPUTimeMS calculates average CPU time in milliseconds
func CalculateAvgCPUTimeMS(totalWorkerTime *int64, executionCount *int64) float64 {
	if totalWorkerTime == nil || executionCount == nil || *executionCount == 0 {
		return 0.0
	}
	return float64(*totalWorkerTime) / float64(*executionCount) / 1000.0
}

// CalculateAvgElapsedTimeMS calculates average elapsed time in milliseconds
func CalculateAvgElapsedTimeMS(totalElapsedTime *int64, executionCount *int64) float64 {
	if totalElapsedTime == nil || executionCount == nil || *executionCount == 0 {
		return 0.0
	}
	return float64(*totalElapsedTime) / float64(*executionCount) / 1000.0
}

// CalculateAvgDiskReads calculates average disk reads per execution
func CalculateAvgDiskReads(totalLogicalReads *int64, executionCount *int64) float64 {
	if totalLogicalReads == nil || executionCount == nil || *executionCount == 0 {
		return 0.0
	}
	return float64(*totalLogicalReads) / float64(*executionCount)
}

// CalculateAvgDiskWrites calculates average disk writes per execution
func CalculateAvgDiskWrites(totalLogicalWrites *int64, executionCount *int64) float64 {
	if totalLogicalWrites == nil || executionCount == nil || *executionCount == 0 {
		return 0.0
	}
	return float64(*totalLogicalWrites) / float64(*executionCount)
}

// NewRelicSlowQueryDetails contains only the fields we want to send to New Relic
type NewRelicSlowQueryDetails struct {
	// Metadata attributes
	QueryID                *models.HexString `metric_name:"query_id" source_type:"attribute"`
	QueryPlanHash          *models.HexString `metric_name:"query_plan_hash" source_type:"attribute"`
	PlanHandle             *models.HexString `metric_name:"plan_handle" source_type:"attribute"`
	QueryText              *string           `metric_name:"query_text" source_type:"attribute"`
	DatabaseName           *string           `metric_name:"database_name" source_type:"attribute"`
	SchemaName             *string           `metric_name:"schema_name" source_type:"attribute"`
	LastExecutionTimestamp *string           `metric_name:"last_execution_timestamp" source_type:"attribute"`
	StatementType          *string           `metric_name:"statement_type" source_type:"attribute"`
	CollectionTimestamp    *string           `metric_name:"collection_timestamp" source_type:"attribute"`
	
	// Only execution count (for context) and calculated averages
	ExecutionCount   *int64  `metric_name:"execution_count" source_type:"gauge"`
	AvgCPUTimeMS     float64 `metric_name:"avg_cpu_time_ms" source_type:"gauge"`
	AvgElapsedTimeMS float64 `metric_name:"avg_elapsed_time_ms" source_type:"gauge"`
	AvgDiskReads     float64 `metric_name:"avg_disk_reads" source_type:"gauge"`
	AvgDiskWrites    float64 `metric_name:"avg_disk_writes" source_type:"gauge"`
}

// EnrichedSlowQueryDetails extends TopNSlowQueryDetails with calculated averages (for internal processing)
type EnrichedSlowQueryDetails struct {
	models.TopNSlowQueryDetails
	AvgCPUTimeMS     float64 `metric_name:"avg_cpu_time_ms" source_type:"gauge"`
	AvgElapsedTimeMS float64 `metric_name:"avg_elapsed_time_ms" source_type:"gauge"`
	AvgDiskReads     float64 `metric_name:"avg_disk_reads" source_type:"gauge"`
	AvgDiskWrites    float64 `metric_name:"avg_disk_writes" source_type:"gauge"`
}

// ToNewRelicFormat converts EnrichedSlowQueryDetails to NewRelicSlowQueryDetails (excludes totals)
func (e EnrichedSlowQueryDetails) ToNewRelicFormat() NewRelicSlowQueryDetails {
	return NewRelicSlowQueryDetails{
		QueryID:                e.QueryID,
		QueryPlanHash:          e.QueryPlanHash,
		PlanHandle:             e.PlanHandle,
		QueryText:              e.QueryText,
		DatabaseName:           e.DatabaseName,
		SchemaName:             e.SchemaName,
		LastExecutionTimestamp: e.LastExecutionTimestamp,
		StatementType:          e.StatementType,
		CollectionTimestamp:    e.CollectionTimestamp,
		ExecutionCount:         e.ExecutionCount,
		AvgCPUTimeMS:           e.AvgCPUTimeMS,
		AvgElapsedTimeMS:       e.AvgElapsedTimeMS,
		AvgDiskReads:           e.AvgDiskReads,
		AvgDiskWrites:          e.AvgDiskWrites,
	}
}

// EnrichSlowQueryWithAverages creates an enriched model with calculated average metrics
func EnrichSlowQueryWithAverages(model models.TopNSlowQueryDetails) EnrichedSlowQueryDetails {
	return EnrichedSlowQueryDetails{
		TopNSlowQueryDetails: model,
		AvgCPUTimeMS:         CalculateAvgCPUTimeMS(model.TotalWorkerTime, model.ExecutionCount),
		AvgElapsedTimeMS:     CalculateAvgElapsedTimeMS(model.TotalElapsedTime, model.ExecutionCount),
		AvgDiskReads:         CalculateAvgDiskReads(model.TotalLogicalReads, model.ExecutionCount),
		AvgDiskWrites:        CalculateAvgDiskWrites(model.TotalLogicalWrites, model.ExecutionCount),
	}
}
