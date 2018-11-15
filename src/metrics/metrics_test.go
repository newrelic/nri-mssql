package metrics

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	update = flag.Bool("update", false, "update .golden files")
)

func updateGoldenFile(data []byte, sourceFile string) {
	if *update {
		ioutil.WriteFile(sourceFile, data, 0644)
	}
}

func createTestEntity(t *testing.T) (i *integration.Integration, e *integration.Entity) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}
	e, err = i.Entity("test", "instance")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	return
}

func checkAgainstFile(t *testing.T, data []byte, expectedFile string) {
	expectedData, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Errorf("Could not read expected file: %v", err.Error())
	}

	assert.Equal(t, data, expectedData)
}

func Test_populateDatabaseMetrics(t *testing.T) {
	i, _ := createTestEntity(t)

	conn, mock := connection.CreateMockSQL(t)
	defer conn.Close()

	databaseRows := sqlmock.NewRows([]string{"db_name"}).
		AddRow("master").
		AddRow("otherdb")
	logGrowthRows := sqlmock.NewRows([]string{"db_name", "log_growth"}).
		AddRow("master", 0).
		AddRow("otherdb", 1)

	// only match the performance counter query
	mock.ExpectQuery(`select name as db_name from sys\.databases`).
		WillReturnRows(databaseRows)

	mock.ExpectQuery(`select\s+RTRIM\(t1\.instance_name\).*`).
		WillReturnRows(logGrowthRows)

	mock.ExpectClose()

	PopulateDatabaseMetrics(i, "MSSQL", conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("..", "testdata", "databaseMetrics.json.golden")
	updateGoldenFile(actual, expectedFile)
	checkAgainstFile(t, actual, expectedFile)
}

func Test_dbMetric_Populator_DBNameError(t *testing.T) {
	modelChan := make(chan interface{}, 10)
	var wg sync.WaitGroup

	// Setup
	i, _ := createTestEntity(t)
	masterEntity, err := i.Entity("master", "database")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	metricSet := masterEntity.NewMetricSet("MssqlDatabaseSample",
		metric.Attribute{Key: "displayName", Value: "master"},
		metric.Attribute{Key: "entityName", Value: "database:master"},
	)

	// used to make sure the number of attributes does not change
	expectedNumAttributes := len(metricSet.Metrics)

	lookup := database.DBMetricSetLookup{"master": metricSet}

	model := struct {
		Metric int
	}{
		1,
	}

	wg.Add(1)

	// Test run
	go dbMetricPopulator(lookup, modelChan, &wg)

	modelChan <- model

	close(modelChan)

	// Setup timeout
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		if length := len(metricSet.Metrics); length != expectedNumAttributes {
			t.Errorf("Expected %d attributes got %d", expectedNumAttributes, length)
		}
	case <-time.After(time.Duration(1) * time.Second):
		t.Error("Waitgroup never returned")
		t.FailNow()
	}
}

func Test_populateInstanceMetrics(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := connection.CreateMockSQL(t)
	defer conn.Close()

	perfCounterRows := sqlmock.NewRows([]string{"buffer_cache_hit_ratio", "buffer_pool_hit_percent", "sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"}).
		AddRow(22, 100, 4736, 142, 3, 641, 2509, 848, 0, 67, 0, 18021, 1112946000, 184700, 0)

	// only match the performance counter query
	mock.ExpectQuery(`SELECT\s+t1.cntr_value AS buffer_cache_hit_ratio.*`).WillReturnRows(perfCounterRows)
	mock.ExpectClose()

	PopulateInstanceMetrics(e, conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("..", "testdata", "perfCounter.json.golden")
	updateGoldenFile(actual, expectedFile)

	checkAgainstFile(t, actual, expectedFile)
}

func Test_populateInstanceMetrics_NoReturn(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := connection.CreateMockSQL(t)
	defer conn.Close()

	perfCounterRows := sqlmock.NewRows([]string{"buffer_cache_hit_ratio", "buffer_pool_hit_percent", "sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"})

	// only match the performance counter query
	mock.ExpectQuery(`SELECT\s+t1.cntr_value AS buffer_cache_hit_ratio.*`).WillReturnRows(perfCounterRows)
	mock.ExpectClose()

	PopulateInstanceMetrics(e, conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("..", "testdata", "empty.json.golden")
	updateGoldenFile(actual, expectedFile)

	checkAgainstFile(t, actual, expectedFile)
}

func Test_populateWaitTimeMetrics(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := connection.CreateMockSQL(t)
	defer conn.Close()

	waitTimeRows := sqlmock.NewRows([]string{"wait_type", "wait_time", "waiting_tasks_count"}).
		AddRow("LCK_M_S", 638, 1).
		AddRow("CHKPT", 1142, 1).
		AddRow("LAZYWRITER_SLEEP", 1118786296, 1126388).
		AddRow("PREEMPTIVE_OS_DEVICEOPS", 119, 90)

	// only match the performance counter query
	mock.ExpectQuery(`SELECT wait_type, wait_time_ms AS wait_time, waiting_tasks_count\s*FROM sys.dm_os_wait_stats wait_stats\s*WHERE wait_time_ms != 0`).WillReturnRows(waitTimeRows)
	mock.ExpectClose()

	populateWaitTimeMetrics(e, conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("..", "testdata", "waitTime.json.golden")
	updateGoldenFile(actual, expectedFile)

	checkAgainstFile(t, actual, expectedFile)
}
