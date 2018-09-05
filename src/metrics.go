package main

import (
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

func populateInstanceMetrics(instanceEntity *integration.Entity, connection *SQLConnection) {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
	)

	for _, queryDef := range instanceDefinitions {
		rows := queryDef.GetDataModels()
		if err := connection.Query(rows, queryDef.GetQuery()); err != nil {
			log.Error("Could not execute query: %s", err.Error())
			continue
		}

		vp := reflect.Indirect(reflect.ValueOf(rows))
		vpInterface := vp.Index(0).Interface()
		err := metricSet.MarshalMetrics(vpInterface)
		if err != nil {
			log.Error("Could not parse metrics from query result: %s", err.Error())
		}
	}

	populateWaitTimeMetrics(instanceEntity, connection)
}

func populateWaitTimeMetrics(instanceEntity *integration.Entity, connection *SQLConnection) {
	rows := make([]waitTimeRows, 0)
	if err := connection.Query(&rows, waitTimeQuery); err != nil {
		log.Error("Could not execute query: %s", err.Error())
		return
	}

	for _, row := range rows {
		metricSet := instanceEntity.NewMetricSet("MssqlWaitSample",
			metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
			metric.Attribute{Key: "waitType", Value: *row.WaitType},
		)

		metrics := []struct{
			metricName string
			metricValue int
			metricType metric.SourceType
		}{
			{
				"system.waitTimeCount", *row.WaitCount, metric.GAUGE,
			},
			{
				"system.waitTimeInMillisecondsPerSecond", *row.WaitTime, metric.GAUGE,
			},
		}

		for _, metric := range metrics {
			err := metricSet.SetMetric(metric.metricName, metric.metricValue, metric.metricType)
			if err != nil {
				log.Error("Could not set wait time metric: %s", err.Error())
			}
		}
	}
}

func populateDatabaseMetrics(i *integration.Integration, con *SQLConnection) error {
	// create database entities
	dbEntities, err := createDatabaseEntities(i, con)
	if err != nil {
		return err
	}

	// create database entities lookup for fast metric set
	dbSetLookup := createDBEntitySetLookup(dbEntities)

	modelChan := make(chan interface{}, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go dbMetricPopulator(dbSetLookup, modelChan, &wg)

	// run queries that are not specific to a database
	processGeneralDBDefinitions(con, modelChan)

	// run queries that are specific to a database
	processSpecificDBDefinitions(con, dbSetLookup.GetDBNames(), modelChan)

	close(modelChan)
	wg.Wait()

	return nil
}

func processGeneralDBDefinitions(con *SQLConnection, modelChan chan<- interface{}) {
	for _, queryDef := range databaseDefinitions {
		makeDBQuery(con, queryDef.GetQuery(), queryDef.GetDataModels(), modelChan)
	}
}

func processSpecificDBDefinitions(con *SQLConnection, dbNames []string, modelChan chan<- interface{}) {
	for _, queryDef := range specificDatabaseDefinitions {
		for _, dbName := range dbNames {
			query := queryDef.GetQuery(dbNameReplace(dbName))
			makeDBQuery(con, query, queryDef.GetDataModels(), modelChan)
		}
	}
}

func makeDBQuery(con *SQLConnection, query string, models interface{}, modelChan chan<- interface{}) {
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

func dbMetricPopulator(dbSetLookup DBMetricSetLookup, modelChan <-chan interface{}, wg *sync.WaitGroup) {
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
