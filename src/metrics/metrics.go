// Package metrics contains all the code that is used to collect metrics from the target
package metrics

import (
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
)

// PopulateInstanceMetrics creates instance-level metrics
func PopulateInstanceMetrics(instanceEntity *integration.Entity, connection *connection.SQLConnection) {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
		metric.Attribute{Key: "host", Value: connection.Host},
	)

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

// PopulateDatabaseMetrics collects per-database metrics
func PopulateDatabaseMetrics(i *integration.Integration, instanceName string, connection *connection.SQLConnection) error {
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
