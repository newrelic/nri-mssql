// Package metrics contains all the code that is used to collect metrics from the target
package metrics

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
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

// PopulateInstanceMetrics creates instance-level metrics
func PopulateInstanceMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
		metric.Attribute{Key: "host", Value: connection.Host},
	)

	collectionList := instanceDefinitions
	if arguments.EnableBufferMetrics {
		collectionList = append(collectionList, instanceBufferDefinitions...)
	}

	for _, queryDef := range instanceDefinitions {
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
		populateCustomMetrics(instanceEntity, connection, customQuery{Query: arguments.CustomMetricsQuery})
	} else if len(arguments.CustomMetricsConfig) > 0 {
		queries, err := parseCustomQueries(arguments)
		if err != nil {
			log.Error("Failed to parse custom queries: %s", err)
		}
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
	b, err := ioutil.ReadFile(arguments.CustomMetricsConfig)
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
			metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			metric.Attribute{Key: "waitType", Value: *model.WaitType},
			metric.Attribute{Key: "host", Value: connection.Host},
			metric.Attribute{Key: "instance", Value: instanceEntity.Metadata.Name},
		)

		metrics := []struct {
			metricName  string
			metricValue int
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

	rows, err := connection.Queryx(prefix + query.Query)
	if err != nil {
		log.Error("Could not execute custom query: %s", err)
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			log.Error("Failed to scan custom query row: %s", err)
			return
		}

		nameInterface, ok := row["metric_name"]
		var name string
		if !ok {
			if len(query.Name) > 0 {
				name = query.Name
			}
		} else {
			name, ok = nameInterface.(string)
			if !ok {
				log.Error("Non-string type %T for custom query 'metric_name' column", nameInterface)
				return
			}
		}

		value, ok := row["metric_value"]
		var valueString string
		if !ok {
			if len(name) > 0 {
				log.Error("Missing 'metric_value' for %s in custom query", name)
				return
			}
		} else {
			valueString = fmt.Sprintf("%v", value)
			if len(name) == 0 {
				log.Error("Missing 'metric_name' for %s in custom query", valueString)
				return
			}
		}

		if len(query.Prefix) > 0 {
			name = query.Prefix + name
		}

		var metricType metric.SourceType
		metricTypeInterface, ok := row["metric_type"]
		if !ok {
			if len(query.Type) > 0 {
				metricType, err = metric.SourceTypeForName(query.Type)
				if err != nil {
					log.Error("Invalid metric type %s in YAML: %s", query.Type, err)
					return
				}
			} else {
				metricType = detectMetricType(valueString)
			}
		} else {
			// metric type was specified
			metricTypeString, ok := metricTypeInterface.(string)
			if !ok {
				log.Error("Non-string type %T for custom query 'metric_type' column", metricTypeInterface)
				return
			}
			metricType, err = metric.SourceTypeForName(metricTypeString)
			if err != nil {
				log.Error("Invalid metric type %s in query 'metric_type' column: %s", metricTypeString, err)
				return
			}
		}

		attributes := []metric.Attribute{
			{Key: "displayName", Value: instanceEntity.Metadata.Name},
			{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			{Key: "host", Value: connection.Host},
			{Key: "instance", Value: instanceEntity.Metadata.Name},
		}
		if len(query.Database) > 0 {
			attributes = append(attributes, metric.Attribute{Key: "database", Value: query.Database})
		}
		ms := instanceEntity.NewMetricSet("MssqlCustomQuerySample", attributes...)

		for k, v := range row {
			if k == "metric_name" || k == "metric_type" || k == "metric_value" {
				continue
			}
			vString := fmt.Sprintf("%v", v)

			if len(query.Prefix) > 0 {
				k = query.Prefix + k
			}

			err = ms.SetMetric(k, vString, detectMetricType(vString))
			if err != nil {
				log.Error("Failed to set metric: %s", err)
				continue
			}
		}

		if len(valueString) > 0 {
			err = ms.SetMetric(name, valueString, metricType)
			if err != nil {
				log.Error("Failed to set metric: %s", err)
			}
		}
	}
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

func detectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}
