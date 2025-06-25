// Package metrics contains all the code that is used to collect metrics from the target
package metrics

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/common"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
	"gopkg.in/yaml.v2"
)

type customQuery struct {
	Query    string
	Prefix   string
	Name     string `yaml:"metric_name"`
	Type     string `yaml:"metric_type"`
	Database string
}

// customQueryMetricValue represents a metric value fetched from the results of a custom query
type customQueryMetricValue struct {
	value      any
	sourceType metric.SourceType
}

var (
	errMissingMetricValueCustomQuery = errors.New("missing 'metric_value' for custom query")
	errMissingMetricNameCustomQuery  = errors.New("missing 'metric_name' for custom query")
)

const (
	// Maximum number of metrics retrieved from a single query execution
	resultsBufferSizePerWorker = 5
)

// PopulateInstanceMetrics creates instance-level metrics
// The below function has too many if's which is needed , so ignoring the golint error by adding below linter directive.
//
//nolint:gocyclo
func PopulateInstanceMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList, engineEdition int) {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		attribute.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
		attribute.Attribute{Key: "host", Value: connection.Host},
	)

	collectionList := instanceDefinitions
	collectionList = append(collectionList, GetQueryDefinitions(MemoryQueries, engineEdition)...)
	if arguments.EnableBufferMetrics {
		collectionList = append(collectionList, instanceBufferDefinitions...)
	}
	if arguments.EnableDiskMetricsInBytes {
		collectionList = append(collectionList, diskMetricInBytesDefinition...)
	}

	for _, queryDef := range collectionList {
		models := queryDef.GetDataModels()
		if common.SkipQueryForEngineEdition(engineEdition, queryDef.GetQuery()) {
			log.Debug("Skipping query '%s' for unsupported engine edition %d", queryDef.GetQuery(), engineEdition)
			continue
		}
		if err := connection.Query(models, queryDef.GetQuery()); err != nil {
			log.Error("Could not execute instance query: %s", err.Error())
			continue
		}

		vp := reflect.Indirect(reflect.ValueOf(models))

		// Nothing was returned
		if vp.Len() == 0 {
			log.Debug("No data returned from instance query '%s'", queryDef.GetQuery())
			continue
		}

		vpInterface := vp.Index(0).Interface()
		err := metricSet.MarshalMetrics(vpInterface)
		if err != nil {
			log.Error("Could not parse metrics from instance query result: %s", err.Error())
		}
	}

	populateWaitTimeMetrics(instanceEntity, connection)

	if len(arguments.CustomMetricsQuery) > 0 {
		log.Debug("Arguments custom metrics query: %s", arguments.CustomMetricsQuery)
		populateCustomMetrics(instanceEntity, connection, customQuery{Query: arguments.CustomMetricsQuery})
	} else if len(arguments.CustomMetricsConfig) > 0 {
		queries, err := parseCustomQueries(arguments)
		if err != nil {
			log.Error("Failed to parse custom queries: %s", err)
		}
		log.Debug("Parsed custom queries: %+v", queries)
		var wg sync.WaitGroup
		for _, query := range queries {
			wg.Add(1)
			go func(query customQuery) {
				defer wg.Done()
				populateCustomMetrics(instanceEntity, connection, query)
			}(query)
		}
		wg.Wait()
	}
}

func parseCustomQueries(arguments args.ArgumentList) ([]customQuery, error) {
	// load YAML config file
	b, err := os.ReadFile(arguments.CustomMetricsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom_metrics_config: %s", err)
	}
	// parse
	var c struct{ Queries []customQuery }
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse custom_metrics_config: %s", err)
	}

	return c.Queries, nil
}

func populateWaitTimeMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection) {
	models := make([]waitTimeModel, 0)
	if err := connection.Query(&models, waitTimeQuery); err != nil {
		log.Error("Could not execute query: %s", err.Error())
		return
	}

	for _, model := range models {
		metricSet := instanceEntity.NewMetricSet("MssqlWaitSample",
			attribute.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			attribute.Attribute{Key: "waitType", Value: *model.WaitType},
			attribute.Attribute{Key: "host", Value: connection.Host},
			attribute.Attribute{Key: "instance", Value: instanceEntity.Metadata.Name},
		)

		metrics := []struct {
			metricName  string
			metricValue int64
			metricType  metric.SourceType
		}{
			{
				"system.waitTimeCount", *model.WaitCount, metric.GAUGE,
			},
			{
				"system.waitTimeInMillisecondsPerSecond", *model.WaitTime, metric.GAUGE,
			},
		}

		for _, metric := range metrics {
			err := metricSet.SetMetric(metric.metricName, metric.metricValue, metric.metricType)
			if err != nil {
				log.Error("Could not set wait time metric '%s' for wait type '%s': %s", metric.metricName, model.WaitType, err.Error())
			}
		}
	}
}

// Execute one or more custom queries
func populateCustomMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection, query customQuery) {
	var prefix string
	if len(query.Database) > 0 {
		prefix = "USE " + query.Database + "; "
	}

	log.Debug("Running custom query: %+v", query)

	rows, err := connection.Queryx(prefix + query.Query)
	if err != nil {
		log.Error("Could not execute custom query: %s", err)
		return
	}
	columns, err := rows.Columns()
	if err != nil {
		log.Error("Could not fetch types information from custom query", err)
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	var rowCount = 0
	for rows.Next() {
		rowCount++
		values := make([]sql.NullString, len(columns))         // All values are represented as null strings (the corresponding conversion is handled while scanning)
		valuesForScanning := make([]interface{}, len(columns)) // the `rows.Scan` function requires an array of interface{}
		for i := range valuesForScanning {
			valuesForScanning[i] = &values[i]
		}
		if err := rows.Scan(valuesForScanning...); err != nil {
			log.Error("Failed to scan custom query row: %s", err)
			return
		}

		attributes := []attribute.Attribute{
			{Key: "displayName", Value: instanceEntity.Metadata.Name},
			{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			{Key: "host", Value: connection.Host},
			{Key: "instance", Value: instanceEntity.Metadata.Name},
		}
		if len(query.Database) > 0 {
			attributes = append(attributes, attribute.Attribute{Key: "database", Value: query.Database})
		}
		ms := instanceEntity.NewMetricSet("MssqlCustomQuerySample", attributes...)

		dbMetrics, err := metricsFromCustomQueryRow(values, columns, query)
		if err != nil {
			log.Error("Error fetching metrics from query %s (query: %s)", err, query.Query)
		}
		for name, dbMetric := range dbMetrics {
			err = ms.SetMetric(name, dbMetric.value, dbMetric.sourceType)
			if err != nil {
				log.Error("Failed to set metric: %s", err)
				continue
			}
		}
	}

	if rowCount == 0 {
		log.Warn("No result set found for custom query: %+v", query)
	} else {
		log.Debug("%v Rows returned for custom query: %+v", rowCount, query)
	}

	if err := rows.Err(); err != nil {
		log.Error("Error iterating rows: %s", err)
	}
}

// metricsFromCustomQueryRow obtains a map of metrics from a row resulting from a custom query.
// A particular metric can be configured either with:
// - Specific columns in the query: metric_name, metric_type, metric_value
// - The corresponding `query.Name` and `query.Type`
// When both are defined, the query columns have precedence. Besides, if type is not defined it is automatically deteced.
// The rest of the query columns are also taken as metrics/attributes (detecting their types automatically).
// Besides, if `query.Prefix` is defined, all metric and attribute names will include the corresponding prefix.
func metricsFromCustomQueryRow(row []sql.NullString, columns []string, query customQuery) (map[string]customQueryMetricValue, error) {
	metrics := map[string]customQueryMetricValue{}

	var metricValue string
	metricType := query.Type
	metricName := query.Name

	for i, columnName := range columns { // Scan the query columns to extract the corresponding metrics
		elementValue := extractValue(row[i])
		switch columnName {
		// Handle columns with 'special' meaning
		case "metric_name":
			metricName = elementValue
		case "metric_type":
			metricType = elementValue
		case "metric_value":
			metricValue = elementValue
		// The rest of the values are taken as metrics/attributes with automatically detected type.
		default:
			name := query.Prefix + columnName
			// value is passed as empty string if row[i] value is nil
			value := ""
			if row[i].Valid {
				value = row[i].String
			}
			metrics[name] = customQueryMetricValue{value: value, sourceType: DetectMetricType(value)}
		}
	}

	customQueryMetric, err := metricFromTargetColumns(metricValue, metricName, metricType, query)
	if err != nil {
		return nil, fmt.Errorf("could not extract metric from query: %w", err)
	}
	if customQueryMetric != nil {
		metricName = query.Prefix + metricName
		metrics[metricName] = *customQueryMetric
	}
	return metrics, nil
}

/*
In Order to handle null values in the output of custom query the extractValue function is used.

The extractValue checks if the given sql.NullString is not null.
  - If not null it returns the string value
  - Otherwise it returns empty string
*/
func extractValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// metricFromTargetColumns builds a customQueryMetricValue from the values in target columns (or defaults in the yaml
// configuration). It returns an error if values are inconsistent (Ex: metricName is set but metricValue is not) and it
// can be nil the metric was not defined.
func metricFromTargetColumns(metricValue, metricName, metricType string, query customQuery) (*customQueryMetricValue, error) {
	if metricValue == "" {
		if metricName != "" {
			return nil, fmt.Errorf("%w: name %q, query %q", errMissingMetricValueCustomQuery, metricName, query.Query)
		}
		return nil, nil // Ignored when there is no value and no name
	}

	if metricName == "" {
		return nil, fmt.Errorf("%w: query %q", errMissingMetricNameCustomQuery, query.Query)
	}

	var sourceType metric.SourceType
	if metricType != "" {
		sourceTypeFromQuery, err := metric.SourceTypeForName(metricType)
		if err != nil {
			return nil, fmt.Errorf("invalid metric type %s: %w", metricType, err)
		}
		sourceType = sourceTypeFromQuery
	} else {
		sourceType = DetectMetricType(metricValue)
	}

	return &customQueryMetricValue{value: metricValue, sourceType: sourceType}, nil
}

type databaseMetricsProcessor func(*integration.Integration, string, *connection.SQLConnection, args.ArgumentList, database.DBMetricSetLookup, int, chan<- interface{})

// Bucket for processor functions
var processorFunctionSet = EngineSet[databaseMetricsProcessor]{
	Default:                 processDefaultDBMetrics,
	AzureSQLDatabase:        processAzureSQLDatabaseMetrics,
	AzureSQLManagedInstance: processDefaultDBMetrics,
}

// PopulateDatabaseMetrics collects per-database metrics
func PopulateDatabaseMetrics(i *integration.Integration, instanceName string, connection *connection.SQLConnection, arguments args.ArgumentList, engineEdition int) error {
	// create database entities
	dbEntities, err := database.CreateDatabaseEntities(i, connection, instanceName)
	if err != nil {
		return err
	}

	// create database entities lookup for fast metric set
	dbSetLookup := database.CreateDBEntitySetLookup(dbEntities, instanceName, connection.Host)

	maxWorkers := arguments.GetMaxConcurrentWorkers()

	// A buffer sized to allow each worker to offload a burst of results without blocking the worker pool.
	modelChan := make(chan interface{}, maxWorkers*resultsBufferSizePerWorker)
	var wg sync.WaitGroup

	wg.Add(1)
	go dbMetricPopulator(dbSetLookup, modelChan, &wg)

	processor := processorFunctionSet.Select(engineEdition)
	processor(i, instanceName, connection, arguments, dbSetLookup, engineEdition, modelChan)

	close(modelChan)
	wg.Wait()

	return nil
}

// processDefaultDBMetrics handles metric collection for a standard SQL Server instance.
func processDefaultDBMetrics(i *integration.Integration, instanceName string, connection *connection.SQLConnection, arguments args.ArgumentList, dbSetLookup database.DBMetricSetLookup, engineEdition int, modelChan chan<- interface{}) {
	// run queries that are not specific to a database
	processDBDefinitions(connection, GetQueryDefinitions(StandardQueries, engineEdition), modelChan)

	// run queries that are not specific to a database
	if arguments.EnableBufferMetrics {
		processDBBufferDefinitions(connection, modelChan)
	}

	// run queries that are specific to a database
	if arguments.EnableDatabaseReserveMetrics {
		processSpecificDBDefinitions(connection, dbSetLookup.GetDBNames(), modelChan)
	}
}

// processAzureSQLDatabaseMetrics handles metric collection for Azure SQL Database concurrently.
// It dispatches the work of processing each database to a worker goroutine.
func processAzureSQLDatabaseMetrics(i *integration.Integration, instanceName string, _ *connection.SQLConnection, arguments args.ArgumentList, dbSetLookup database.DBMetricSetLookup, engineEdition int, modelChan chan<- interface{}) {
	databaseNames := dbSetLookup.GetDBNames()

	maxWorkers := arguments.GetMaxConcurrentWorkers()
	dbChan := make(chan struct{}, maxWorkers)
	var waitGroup sync.WaitGroup

	for _, dbName := range databaseNames {
		waitGroup.Add(1)
		dbChan <- struct{}{}
		go processSingleAzureDB(&waitGroup, dbChan, dbName, arguments, engineEdition, modelChan)
	}
	waitGroup.Wait()
}

func processSingleAzureDB(wg *sync.WaitGroup, dbChan chan struct{}, dbName string, arguments args.ArgumentList, engineEdition int, modelChan chan<- interface{}) {
	defer wg.Done()
	defer func() { <-dbChan }()

	con, err := connection.CreateDatabaseConnection(&arguments, dbName)
	if err != nil {
		log.Error("Error creating connection to SQL Server: %s", err.Error())
		log.Warn("Skipping populating db metrics for database : %s", dbName)
		return
	}
	defer con.Close()

	processDBDefinitions(con, GetQueryDefinitions(StandardQueries, engineEdition), modelChan)

	processMemoryDBDefinitions(con, dbName, modelChan)

	if arguments.EnableDiskMetricsInBytes {
		processDBDefinitions(con, databaseDiskDefinitionsForAzureSQLDatabase, modelChan)
	}

	if arguments.EnableBufferMetrics {
		processDBDefinitions(con, GetQueryDefinitions(BufferQueries, engineEdition), modelChan)
	}

	if arguments.EnableDatabaseReserveMetrics {
		processDBDefinitions(con, GetQueryDefinitions(SpecificQueries, engineEdition), modelChan)
	}
}

func processMemoryDBDefinitions(con *connection.SQLConnection, dbName string, modelChan chan<- interface{}) {
	var memUtilResult []*MemoryUtilizationModel
	if err := con.Query(&memUtilResult, memoryUtilizationQuery); err != nil {
		log.Error("Encountered the following error: %s. Running query '%s'", err.Error(), memoryUtilizationQuery)
	} else {
		sendModelsToPopulator(modelChan, memUtilResult)
	}

	var totalMemResult []*TotalPhysicalMemoryModel
	if err := con.Query(&totalMemResult, totalPhysicalMemoryQuery); err != nil {
		log.Error("Encountered the following error: %s. Running query '%s'", err.Error(), totalPhysicalMemoryQuery)
	} else {
		sendModelsToPopulator(modelChan, totalMemResult)
	}

	if len(memUtilResult) > 0 && memUtilResult[0].MemoryUtilization != nil &&
		len(totalMemResult) > 0 && totalMemResult[0].TotalPhysicalMemory != nil {
		utilization := *memUtilResult[0].MemoryUtilization
		totalMemory := *totalMemResult[0].TotalPhysicalMemory

		memoryAvailable := (math.Abs((100.0 - utilization)) / 100) * totalMemory

		availableModel := &AvailablePhysicalMemoryModel{
			DataModel:               database.DataModel{DBName: dbName},
			AvailablePhysicalMemory: &memoryAvailable,
		}
		sendModelsToPopulator(modelChan, []*AvailablePhysicalMemoryModel{availableModel})
	} else {
		log.Debug("Could not calculate memoryAvailable due to missing memoryUtilization or memoryTotal metrics.")
	}
}

func processDBDefinitions(con *connection.SQLConnection, definitions []*QueryDefinition, modelChan chan<- interface{}) {
	for _, queryDef := range definitions {
		makeDBQuery(con, queryDef.GetQuery(), queryDef.GetDataModels(), modelChan)
	}
}

func processDBBufferDefinitions(con *connection.SQLConnection, modelChan chan<- interface{}) {
	for _, queryDef := range databaseBufferDefinitions {
		makeDBQuery(con, queryDef.GetQuery(), queryDef.GetDataModels(), modelChan)
	}
}

func processSpecificDBDefinitions(con *connection.SQLConnection, dbNames []string, modelChan chan<- interface{}) {
	for _, queryDef := range specificDatabaseDefinitions {
		for _, dbName := range dbNames {
			query := queryDef.GetQuery(dbNameReplace(dbName))
			makeDBQuery(con, query, queryDef.GetDataModels(), modelChan)
		}
	}
}

func makeDBQuery(con *connection.SQLConnection, query string, models interface{}, modelChan chan<- interface{}) {
	if err := con.Query(models, query); err != nil {
		log.Error("Encountered the following error: %s. Running query '%s'", err.Error(), query)
		return
	}

	// Send models off to populator
	sendModelsToPopulator(modelChan, models)
}

func sendModelsToPopulator(modelChan chan<- interface{}, models interface{}) {
	v := reflect.ValueOf(models)
	vp := reflect.Indirect(v)

	// because all data models are hard coded we can ensure they are all slices and not type check
	for i := 0; i < vp.Len(); i++ {
		modelChan <- vp.Index(i).Interface()
	}
}

func dbMetricPopulator(dbSetLookup database.DBMetricSetLookup, modelChan <-chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		model, ok := <-modelChan
		if !ok {
			return
		}

		metricSet, ok := dbSetLookup.MetricSetFromModel(model)
		if !ok {
			log.Error("Unable to determine database name, %+v", model)
			continue
		}

		if err := metricSet.MarshalMetrics(model); err != nil {
			log.Error("Error setting database metrics: %s", err.Error())
		}
	}
}

func DetectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}
