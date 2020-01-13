// Package metrics contains all the code that is used to collect metrics from the target
package metrics

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
)

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
	if arguments.CustomMetricsQuery != "" {
		populateCustomMetrics(instanceEntity, connection, arguments.CustomMetricsQuery)
	}
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

func populateCustomMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection, query string) {

	rows, err := connection.Queryx(query)
	if err != nil {
		log.Error("Could not execute custom query: %s", err.Error())
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
		if !ok {
			log.Error("Missing required column 'metric_name' in custom query")
			return
		}
		name, ok := nameInterface.(string)
		if !ok {
			log.Error("Non-string type %T for custom query 'metric_name' column", nameInterface)
			continue
		}

		metricTypeInterface, ok := row["metric_type"]
		if !ok {
			log.Error("Missing required column 'metric_type' in custom query")
			return
		}
		metricTypeString, ok := metricTypeInterface.(string)
		if !ok {
			log.Error("Non-string type %T for custom query 'metric_type' column", metricTypeInterface)
			continue
		}
		metricType, err := metric.SourceTypeForName(metricTypeString)
		if err != nil {
			log.Error("Invalid metric type %s: %s", metricTypeString, err)
			continue
		}

		value, ok := row["metric_value"]
		if !ok {
			log.Error("Missing required column 'metric_type' in custom query")
			return
		}

		attributes := []metric.Attribute{
			{Key: "displayName", Value: instanceEntity.Metadata.Name},
			{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
		}
		for k, v := range row {
			if k == "metric_name" || k == "metric_type" || k == "metric_value" {
				continue
			}

			valString := fmt.Sprintf("%v", v)

			attributes = append(attributes, metric.Attribute{Key: k, Value: valString})
		}

		ms := instanceEntity.NewMetricSet("MssqlCustomQuerySample", attributes...)
		err = ms.SetMetric(name, value, metricType)
		if err != nil {
			log.Error("Failed to set metric: %s", err)
			continue
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
	processSpecificDBDefinitions(connection, dbSetLookup.GetDBNames(), modelChan)

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
