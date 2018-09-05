package main

import (
	"testing"
	"io/ioutil"
	"path/filepath"
	"flag"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/infra-integrations-sdk/integration"
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

func Test_populateMetrics(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := createMockSQL(t)
	defer conn.Close()

	perfCounterRows := sqlmock.NewRows([]string{"buffer_cache_hit_ratio", "buffer_pool_hit_percent", "sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"}).
		AddRow(22, 100, 4736, 142, 3, 641, 2509, 848, 0, 67, 0, 18021, 1112946000, 184700, 0)
		
	// only match the performance counter query
	mock.ExpectQuery(`select\s+t1.cntr_value as buffer_cache_hit_ratio.*`).WillReturnRows(perfCounterRows)

	populateInstanceMetrics(e, conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("testdata", "perfCounter.json.golden")
	updateGoldenFile(actual, expectedFile)

	checkAgainstFile(t, actual, expectedFile)
}

func Test_populateWaitTimeMetrics(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := createMockSQL(t)
	defer conn.Close()

	waitTimeRows := sqlmock.NewRows([]string{"wait_type", "wait_time", "waiting_tasks_count"}).
		AddRow("LCK_M_S", 638, 1).
		AddRow("CHKPT", 1142, 1).
		AddRow("LAZYWRITER_SLEEP", 1118786296, 1126388).
		AddRow("PREEMPTIVE_OS_DEVICEOPS", 119, 90)
		
	// only match the performance counter query
	mock.ExpectQuery(`SELECT wait_type, wait_time_ms AS wait_time, waiting_tasks_count\s*FROM sys.dm_os_wait_stats wait_stats\s*WHERE wait_time_ms != 0`).WillReturnRows(waitTimeRows)

	populateWaitTimeMetrics(e, conn)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("testdata", "waitTime.json.golden")
	updateGoldenFile(actual, expectedFile)

	checkAgainstFile(t, actual, expectedFile)
}