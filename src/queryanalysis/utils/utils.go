package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/instance"
	"github.com/newrelic/nri-mssql/src/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
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
	// 2. Numeric sequences (integers and decimals), such as 123, 456.78, or .99.
	// 3. Double-quoted strings, such as "example".
	// This regex can be useful for identifying and potentially anonymizing literal values
	// in a given text, like extracting or concealing specific data within strings.
	literalAnonymizer = regexp.MustCompile(`'[^']*'|\d*\.?\d+|".*?"`)
	// dmvCommentRemover removes DMV comments like /* DMV_POP_1761636289952111000_85288 */ from the beginning of queries
	dmvCommentRemover = regexp.MustCompile(`^\s*/\*\s*DMV_[^*]*\*/\s*`)
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
	return fmt.Sprintf(query, args.QueryMonitoringFetchInterval, args.QueryMonitoringCountThreshold,
		args.QueryMonitoringResponseTimeThreshold, config.TextTruncateLimit)
}

// formatWaitAnalysis formats the wait analysis query.
// Updated for the simplified query with fixed TOP value (no parameters needed)
func formatWaitAnalysis(query string, args args.ArgumentList) string {
	return query
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
	rows, err := sqlConnection.Connection.Queryx(queryDetailsDto.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()
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

	// Special handling for waitAnalysis to collect all results first, then sort and filter
	if queryDetailsDto.Type == "waitAnalysis" {
		waitResults := make([]models.WaitTimeAnalysis, 0)

		for rows.Next() {
			var model models.WaitTimeAnalysis
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan waitAnalysis row: ", err)
				continue
			}

			waitResults = append(waitResults, model)
		}

		// Sort and filter to get top N by wait time (using QueryMonitoringCountThreshold)
		topWaitResults := sortAndFilterWaitAnalysis(waitResults, arguments.QueryMonitoringCountThreshold)

		// Now process query text for only the top N results: remove DMV comments first, then anonymize
		for i := range topWaitResults {
			if topWaitResults[i].QueryText != nil {
				// Step 1: Remove DMV comments
				cleanedQuery := RemoveDMVComments(*topWaitResults[i].QueryText)
				// Step 2: Anonymize literals
				*topWaitResults[i].QueryText = AnonymizeQueryText(cleanedQuery)
			}

			results = append(results, topWaitResults[i])
		}

		return results, queryIDs, nil
	}

	// Original logic for other query types
	for rows.Next() {
		switch queryDetailsDto.Type {
		case "slowQueries":
			var model models.TopNSlowQueryDetails
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan row: ", err)
				continue
			}
			if model.QueryText != nil {
				// Only anonymize literals for slow queries
				*model.QueryText = AnonymizeQueryText(*model.QueryText)
			}
			results = append(results, model)

			// Collect query IDs for fetching executionPlans
			if model.QueryID != nil {
				queryIDs = append(queryIDs, *model.QueryID)
			}
		case "blockingSessions":
			var model models.BlockingSessionQueryDetails
			if err := rows.StructScan(&model); err != nil {
				log.Debug("Could not scan row: ", err)
				continue
			}
			if model.BlockingQueryText != nil {
				// For blocking sessions, just anonymize (no DMV comments expected)
				*model.BlockingQueryText = AnonymizeQueryText(*model.BlockingQueryText)
			}
			if model.BlockedQueryText != nil {
				// For blocking sessions, just anonymize (no DMV comments expected)
				*model.BlockedQueryText = AnonymizeQueryText(*model.BlockedQueryText)
			}
			results = append(results, model)
		default:
			return nil, queryIDs, fmt.Errorf("%w: %s", ErrUnknownQueryType, queryDetailsDto.Type)
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
		// Only anonymize literals for execution plans
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
		// Convert the result into a map[string]interface{} for dynamic key-value access
		resultMap, err := convertResultToMap(result)
		if err != nil {
			log.Error("failed to convert result: %v", err)
			continue
		}

		// Create a new metric set with the query name and required attributes
		metricSet := instanceEntity.NewMetricSet(queryDetailsDto.EventName,
			attribute.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			attribute.Attribute{Key: "host", Value: sqlConnection.Host},
			attribute.Attribute{Key: "reportingEndpoint", Value: sqlConnection.Host},
		)

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
		log.Error("IngestQueryMetrics - Failed to publish metrics: %v", err)
		return err
	}

	return nil
}

func AnonymizeQueryText(query string) string {
	// Anonymize literals only - this is a generic function
	anonymizedQuery := literalAnonymizer.ReplaceAllString(query, "?")
	return anonymizedQuery
}

// RemoveDMVComments removes DMV comments like /* DMV_POP_1761636289952111000_85288 */ from the beginning of query text
func RemoveDMVComments(query string) string {
	return dmvCommentRemover.ReplaceAllString(query, "")
}

// sortAndFilterWaitAnalysis sorts wait analysis results by TotalWaitTimeMs descending and takes top N records
func sortAndFilterWaitAnalysis(waitResults []models.WaitTimeAnalysis, maxResults int) []models.WaitTimeAnalysis {
	// Handle case where maxResults is 0 or negative - return all results
	if maxResults <= 0 {
		maxResults = len(waitResults)
	}

	// Sort by TotalWaitTimeMs descending
	sort.Slice(waitResults, func(i, j int) bool {
		if waitResults[i].TotalWaitTimeMs == nil && waitResults[j].TotalWaitTimeMs == nil {
			return false
		}
		if waitResults[i].TotalWaitTimeMs == nil {
			return false
		}
		if waitResults[j].TotalWaitTimeMs == nil {
			return true
		}
		return *waitResults[i].TotalWaitTimeMs > *waitResults[j].TotalWaitTimeMs
	})

	// Take top N or all if less than N
	if len(waitResults) < maxResults {
		maxResults = len(waitResults)
	}

	topResults := waitResults[:maxResults]

	return topResults
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
