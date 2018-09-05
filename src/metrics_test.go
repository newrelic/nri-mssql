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

func Test_populateMetrics(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("test", "instance")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

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

	expectedData, _ := ioutil.ReadFile(expectedFile)

	assert.Equal(t, actual, expectedData)
}