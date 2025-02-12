// Package metrics contains all the code that is used to collect metrics from the target
package metrics

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
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

var errMissingMetricValueCustomQuery = errors.New("missing 'metric_value' for custom query")
var errMissingMetricNameCustomQuery = errors.New("missing 'metric_name' for custom query")

// PopulateInstanceMetrics creates instance-level metrics
func PopulateInstanceMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		attribute.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		attribute.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
		attribute.Attribute{Key: "host", Value: connection.Host},
	)

	collectionList := instanceDefinitions
	if arguments.EnableBufferMetrics {
		collectionList = append(collectionList, instanceBufferDefinitions...)
	}
	if arguments.EnableDiskMetricsInBytes {
		collectionList = append(collectionList, diskMetricInBytesDefination...)
	}

	for _, queryDef := range collectionList {
		models := queryDef.GetDataModels()
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
		values := make([]string, len(columns))                 // All values are represented as strings (the corresponding conversion is handled while scanning)
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
func metricsFromCustomQueryRow(row []string, columns []string, query customQuery) (map[string]customQueryMetricValue, error) {
	metrics := map[string]customQueryMetricValue{}

	var metricValue string
	metricType := query.Type
	metricName := query.Name

	for i, columnName := range columns { // Scan the query columns to extract the corresponding metrics
		switch columnName {
		// Handle columns with 'special' meaning
		case "metric_name":
			metricName = row[i]
		case "metric_type":
			metricType = row[i]
		case "metric_value":
			metricValue = row[i]
		// The rest of the values are taken as metrics/attributes with automatically detected type.
		default:
			name := query.Prefix + columnName
			value := row[i]
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

// PopulateDatabaseMetrics collects per-database metrics
func PopulateDatabaseMetrics(i *integration.Integration, instanceName string, connection *connection.SQLConnection, arguments args.ArgumentList) error {
	// create database entities
	dbEntities, err := database.CreateDatabaseEntities(i, connection, instanceName)
	if err != nil {
		return err
	}

	// create database entities lookup for fast metric set
	dbSetLookup := database.CreateDBEntitySetLookup(dbEntities, instanceName, connection.Host)

	modelChan := make(chan interface{}, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go dbMetricPopulator(dbSetLookup, modelChan, &wg)

	// run queries that are not specific to a database
	processGeneralDBDefinitions(connection, modelChan)

	// run queries that are not specific to a database
	if arguments.EnableBufferMetrics {
		processDBBufferDefinitions(connection, modelChan)
	}

	// run queries that are specific to a database
	if arguments.EnableDatabaseReserveMetrics {
		processSpecificDBDefinitions(connection, dbSetLookup.GetDBNames(), modelChan)
	}

	close(modelChan)
	wg.Wait()

	return nil
}

func processGeneralDBDefinitions(con *connection.SQLConnection, modelChan chan<- interface{}) {
	for _, queryDef := range databaseDefinitions {
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
