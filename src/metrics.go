package main

import (
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

func populateInstanceMetrics(instanceEntity *integration.Entity, connection *SQLConnection) error {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
	)

	for _, queryDef := range instanceDefinitions {
		rows := queryDef.GetDataModels()
		if err := connection.Query(rows, queryDef.GetQuery()); err != nil {
			return err
		}

		vp := reflect.Indirect(reflect.ValueOf(rows))
		vpInterface := vp.Index(0).Interface()
		err := metricSet.MarshalMetrics(vpInterface)
		if err != nil {
			return err
		}
	}

	return nil
}

func populateDatabaseMetrics(i *integration.Integration, con *SQLConnection) error {
	dbEntities, err := createDatabaseEntities(i, con)
	if err != nil {
		return err
	}

	dbSetLookup := createDBEntitySetLookup(dbEntities)

	modelChan := make(chan interface{}, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go dbMetricPopulator(dbSetLookup, modelChan, &wg)

	for _, queryDef := range databaseDefinitions {
		wg.Add(1)
		go dbQuerier(con, queryDef, modelChan, &wg)
	}

	wg.Wait()

	return nil
}

func dbQuerier(con *SQLConnection, queryDef *QueryDefinition, modelChan chan<- interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	models := queryDef.GetDataModels()
	if err := con.Query(models, queryDef.GetQuery()); err != nil {
		log.Error("Encountered the following error: %s. Running query '%s'", err.Error(), queryDef.GetQuery())
		return
	}

	// Send models off to populator
	feedModelsDownChannel(modelChan, models)
}

func feedModelsDownChannel(modelChan chan<- interface{}, models interface{}) {
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
			log.Error("Unable to determine database name")
			continue
		}

		if err := metricSet.MarshalMetrics(model); err != nil {
			log.Error("Error setting database metrics: %s", err.Error())
		}
	}
}
