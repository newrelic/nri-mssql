package main

import (
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)
/*
const perfCounterQuery = `select 
t1.cntr_value as buffer_cache_hit_ratio,
(t1.cntr_value * 1.0 / t2.cntr_value) * 100.0 as buffer_pool_hit_percent,
t3.cntr_value as sql_compilations,
t4.cntr_value as sql_recompilations,
t5.cntr_value as user_connections,
t6.cntr_value as lock_wait_time_ms,
t7.cntr_value as page_splits_sec,
t8.cntr_value as checkpoint_pages_sec,
t9.cntr_value as deadlocks_sec,
t10.cntr_value as user_errors,
t11.cntr_value as kill_connection_errors,
t12.cntr_value as batch_request_sec,
(t13.cntr_value * 1000.0) as page_life_expectancy_ms,
t14.cntr_value as transactions_sec,
t15.cntr_value as forced_parameterizations_sec
from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio') t1,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio base') t2,
(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Compilations/sec') t3,
(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Re-Compilations/sec') t4,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'User Connections') t5,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Lock Wait Time (ms)' AND instance_name = '_Total') t6,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Page Splits/sec') t7,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Checkpoint pages/sec') t8,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total') t9,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name = 'User Errors') t10,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name like 'Kill Connection Errors%') t11,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Batch Requests/sec') t12,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13,
(SELECT SUM(cntr_value) as cntr_value FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Transactions/sec') t14,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Forced Parameterizations/sec') t15`

// PerfCounterRow struct for holding the perfCounterQuery results
type PerfCounterRow struct {
	BufferCacheHitRatio *int `db:"buffer_cache_hit_ratio" metric_name:"buffer.cacheHitRatio" source_type:"gauge"`
	BufferPoolHitPercent *float64 `db:"buffer_pool_hit_percent" metric_name:"system.bufferPoolHit" source_type:"gauge"`
	SQLCompilations *int `db:"sql_compilations" metric_name:"stats.sqlCompilationsPerSecond" source_type:"rate"`
	SQLRecompilations *int `db:"sql_recompilations" metric_name:"stats.sqlRecompilationsPerSecond" source_type:"rate"`
	UserConnections *int `db:"user_connections" metric_name:"stats.connections" source_type:"gauge"`
	LockWaitTimeMs *int `db:"lock_wait_time_ms" metric_name:"stats.lockWaitsPerSecond" source_type:"gauge"`
	PageSplitsSec *int `db:"page_splits_sec" metric_name:"access.pageSplitsPerSecond" source_type:"gauge"`
	CheckpointPagesSec *int `db:"checkpoint_pages_sec" metric_name:"buffer.checkpointPagesPerSecond" source_type:"gauge"`
	DeadlocksSec *int `db:"deadlocks_sec" metric_name:"stats.deadlocksPerSecond" source_type:"gauge"`
	UserErrors *int `db:"user_errors" metric_name:"stats.userErrorsPerSecond" source_type:"rate"`
	KillConnectionErrors *int `db:"kill_connection_errors" metric_name:"stats.killConnectionErrorsPerSecond" source_type:"rate"`
	BatchRequestSec *int `db:"batch_request_sec" metric_name:"bufferpool.batchRequestsPerSecond" source_type:"gauge"`
	PageLifeExpectancySec *float64 `db:"page_life_expectancy_ms" metric_name:"bufferpool.pageLifeExpectancyInMilliseconds" source_type:"gauge"`
	TransactionsSec *int `db:"transactions_sec" metric_name:"instance.transactionsPerSecond" source_type:"gauge"`
	ForcedParameterizationsSec *int `db:"forced_parameterizations_sec" metric_name:"instance.forcedParameterizationsPerSecond" source_type:"gauge"`
}
*/
func populateMetrics(instanceEntity *integration.Entity, connection *SQLConnection) error {
	metricSet := instanceEntity.NewMetricSet("MssqlInstanceSample",
		metric.Attribute{Key: "displayName", Value: instanceEntity.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: instanceEntity.Metadata.Namespace + ":" + instanceEntity.Metadata.Name},
	)

	for _, queryDef := range instanceDefinitions {
		rows := queryDef.GetDataModels()
		if err := connection.Query(&rows, queryDef.GetQuery()); err != nil {
			return err
		}

		err := metricSet.MarshalMetrics(rows[0])
		if err != nil {
			return err
		}
	}

	return nil
}

func populateInventoryMetrics(instanceEntity *integration.Entity, connection *SQLConnection) {

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
