package metrics

import (
	"database/sql"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/database"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	update                  = flag.Bool("update", false, "update .golden files")
	errCreateConnectionToDB = errors.New("couldn't create connection to db")
)

func updateGoldenFile(data []byte, sourceFile string) error {
	if *update {
		return os.WriteFile(sourceFile, data, 0600)
	}
	return nil
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
	expectedData, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Errorf("Could not read expected file: %v", err.Error())
	}

	assert.JSONEq(t, string(expectedData), string(data))
}

// Mock connection and args for testing
type mockSQLConnection struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
}

// Mock NewDatabaseConnection
func newMockDatabaseConnection() (*mockSQLConnection, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}

	conn := &mockSQLConnection{db: db, mock: mock}
	return conn, nil
}

func getNewDatabaseConnection1(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
	mockConn, err := newMockDatabaseConnection()
	if err != nil {
		return nil, err
	}

	var metricValue int
	if dbName == "db-1" {
		metricValue = 0
	} else {
		metricValue = 1
	}

	logGrowthRows := sqlmock.NewRows([]string{"db_name", "log_growth"}).
		AddRow(dbName, metricValue)
	ioStallsMetricsRows := sqlmock.NewRows([]string{"db_name", "io_stalls"}).
		AddRow(dbName, metricValue)
	bufferMetricsRows := sqlmock.NewRows([]string{"db_name", "buffer_pool_size"}).
		AddRow(dbName, metricValue)
	spaceMetricsRows := sqlmock.NewRows([]string{"db_name", "reserved_space", "reserved_space_not_used"}).
		AddRow(dbName, metricValue, metricValue)

	mockConn.mock.ExpectQuery(`^SELECT\s+sd\.name\s+AS\s+db_name,\s+spc\.cntr_value\s+AS\s+log_growth\s+FROM\s+sys\.dm_os_performance_counters\s+spc\s+INNER\s+JOIN\s+sys\.databases\s+sd\s+ON\s+sd\.physical_database_name\s+=\s+spc\.instance_name\s+WHERE\s+spc\.counter_name\s+=\s+'Log Growths'\s+AND\s+spc\.object_name\s+LIKE\s+'%:Databases%'\s+AND\s+sd\.database_id\s+=\s+DB_ID\(\)$`).
		WillReturnRows(logGrowthRows)

	mockConn.mock.ExpectQuery(`^*\s+SUM\(io_stall\)\s+AS\s+io_stalls\s+FROM\s+sys\.dm_io_virtual_file_stats\(NULL,\s+NULL\)\s+WHERE\s+database_id\s+=\s+DB_ID\(\)$`).
		WillReturnRows(ioStallsMetricsRows)

	mockConn.mock.ExpectQuery(`^*\s+COUNT_BIG\(\*\)\s+\*\s+\(8\s+\*\s+1024\)\s+AS\s+buffer_pool_size\s+FROM\s+sys\.dm_os_buffer_descriptors\s+WITH\s+\(NOLOCK\)\s+WHERE\s+database_id\s+=\s+DB_ID\(\)$`).
		WillReturnRows(bufferMetricsRows)

	mockConn.mock.ExpectQuery(`^*\s+sum\(a\.total_pages\)\s+\*\s+8\.0\s+\*\s+1024\s+AS\s+reserved_space,\s+\(sum\(a\.total_pages\)\*8\.0\s+-\s+sum\(a\.used_pages\)\*8\.0\)\s+\*\s+1024\s+AS\s+reserved_space_not_used\s+FROM\s+sys\.partitions\s+p\s+with\s+\(nolock\)\s+INNER\s+JOIN\s+sys\.allocation_units\s+a\s+WITH\s+\(NOLOCK\)\s+ON\s+p\.partition_id\s+=\s+a\.container_id\s+LEFT\s+JOIN\s+sys\.internal_tables\s+it\s+WITH\s+\(NOLOCK\)\s+ON\s+p\.object_id\s+=\s+it\.object_id$`).
		WillReturnRows(spaceMetricsRows)

	mockConn.mock.ExpectClose()
	return &connection.SQLConnection{Connection: sqlx.NewDb(mockConn.db, "sqlmock"), Host: "test_host"}, nil
}

func getNewDatabaseConnection2(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
	mockConn, err := newMockDatabaseConnection()
	if err != nil {
		return nil, err
	}

	var metricValue int
	if dbName == "db-1" {
		return nil, errCreateConnectionToDB
	} else {
		metricValue = 1
	}

	logGrowthRows := sqlmock.NewRows([]string{"db_name", "log_growth"}).
		AddRow(dbName, metricValue)
	ioStallsMetricsRows := sqlmock.NewRows([]string{"db_name", "io_stalls"}).
		AddRow(dbName, metricValue)
	spaceMetricsRows := sqlmock.NewRows([]string{"db_name", "reserved_space", "reserved_space_not_used"}).
		AddRow(dbName, metricValue, metricValue)

	mockConn.mock.ExpectQuery(`^SELECT\s+sd\.name\s+AS\s+db_name,\s+spc\.cntr_value\s+AS\s+log_growth\s+FROM\s+sys\.dm_os_performance_counters\s+spc\s+INNER\s+JOIN\s+sys\.databases\s+sd\s+ON\s+sd\.physical_database_name\s+=\s+spc\.instance_name\s+WHERE\s+spc\.counter_name\s+=\s+'Log Growths'\s+AND\s+spc\.object_name\s+LIKE\s+'%:Databases%'\s+AND\s+sd\.database_id\s+=\s+DB_ID\(\)$`).
		WillReturnRows(logGrowthRows)

	mockConn.mock.ExpectQuery(`^*\s+SUM\(io_stall\)\s+AS\s+io_stalls\s+FROM\s+sys\.dm_io_virtual_file_stats\(NULL,\s+NULL\)\s+WHERE\s+database_id\s+=\s+DB_ID\(\)$`).
		WillReturnRows(ioStallsMetricsRows)

	mockConn.mock.ExpectQuery(`^*\s+sum\(a\.total_pages\)\s+\*\s+8\.0\s+\*\s+1024\s+AS\s+reserved_space,\s+\(sum\(a\.total_pages\)\*8\.0\s+-\s+sum\(a\.used_pages\)\*8\.0\)\s+\*\s+1024\s+AS\s+reserved_space_not_used\s+FROM\s+sys\.partitions\s+p\s+with\s+\(nolock\)\s+INNER\s+JOIN\s+sys\.allocation_units\s+a\s+WITH\s+\(NOLOCK\)\s+ON\s+p\.partition_id\s+=\s+a\.container_id\s+LEFT\s+JOIN\s+sys\.internal_tables\s+it\s+WITH\s+\(NOLOCK\)\s+ON\s+p\.object_id\s+=\s+it\.object_id$`).
		WillReturnRows(spaceMetricsRows)

	mockConn.mock.ExpectClose()
	return &connection.SQLConnection{Connection: sqlx.NewDb(mockConn.db, "sqlmock"), Host: "test_host"}, nil
}

func runPopulateDatabaseMetricsTest(
	t *testing.T,
	tc struct {
		name                  string
		setupMock             func(sqlmock.Sqlmock)
		newDatabaseConnection func() func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error)
		args                  args.ArgumentList
		engineEdition         int
		expectedFile          string
		expectError           bool
	},
) {
	i, _ := createTestEntity(t)
	conn, mock := connection.CreateMockSQL(t)
	tc.setupMock(mock)
	// Override NewDatabaseConnection
	originalNewDatabaseConnection := connection.CreateDatabaseConnection
	defer func() {
		connection.CreateDatabaseConnection = originalNewDatabaseConnection
	}()

	connection.CreateDatabaseConnection = tc.newDatabaseConnection()

	assert.NoError(t, PopulateDatabaseMetrics(i, "MSSQL", conn, tc.args, tc.engineEdition))

	actual, _ := i.MarshalJSON()
	assert.NoError(t, updateGoldenFile(actual, tc.expectedFile))
	checkAgainstFile(t, actual, tc.expectedFile)
}

func Test_populateDatabaseMetrics(t *testing.T) {
	// Enable logging if needed
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)

	testCases := []struct {
		name                  string
		setupMock             func(sqlmock.Sqlmock)
		newDatabaseConnection func() func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error)
		args                  args.ArgumentList
		engineEdition         int
		expectedFile          string
		expectError           bool
	}{
		{
			name: "Engine edition 3: Collect metrics",
			setupMock: func(mock sqlmock.Sqlmock) {
				databaseRows := sqlmock.NewRows([]string{"db_name"}).
					AddRow("db-1").
					AddRow("db-2")
				logGrowthRows := sqlmock.NewRows([]string{"db_name", "log_growth"}).
					AddRow("db-1", 0).
					AddRow("db-2", 1)
				bufferMetricsRows := sqlmock.NewRows([]string{"db_name", "buffer_pool_size"}).
					AddRow("db-1", 0).
					AddRow("db-2", 1)

				mock.ExpectQuery(`select name as db_name from sys\.databases`).
					WillReturnRows(databaseRows)

				mock.ExpectQuery(`select\s+RTRIM\(t1\.instance_name\).*`).
					WillReturnRows(logGrowthRows)

				mock.ExpectQuery(`SELECT DB_NAME\(database_id\) AS db_name, buffer_pool_size \* \(8\*1024\) AS buffer_pool_size .*`).WillReturnRows(bufferMetricsRows)

				mock.ExpectClose()
			},
			newDatabaseConnection: func() func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
				return nil
			},
			args: args.ArgumentList{
				EnableBufferMetrics: true,
			},
			engineEdition: 3,
			expectedFile:  filepath.Join("..", "testdata", "databaseMetrics.json.golden"),
			expectError:   false,
		},
		{
			name: "Engine edition 5: Collect all metrics",
			setupMock: func(mock sqlmock.Sqlmock) {
				databaseRows := sqlmock.NewRows([]string{"db_name"}).
					AddRow("db-1").
					AddRow("db-2")

				mock.ExpectQuery(`select name as db_name from sys\.databases`).
					WillReturnRows(databaseRows)

				mock.ExpectClose()
			},
			newDatabaseConnection: func() func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
				return getNewDatabaseConnection1
			},
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: true,
			},
			engineEdition: database.AzureSQLDatabaseEngineEditionNumber,
			expectedFile:  filepath.Join("..", "testdata", "azureSQLDatabaseMetrics.json.golden"),
			expectError:   false,
		},
		{
			name: "Engine edition 5: Error creating connection to db-1, collect db-2 metircs",
			setupMock: func(mock sqlmock.Sqlmock) {
				databaseRows := sqlmock.NewRows([]string{"db_name"}).
					AddRow("db-1").
					AddRow("db-2")

				mock.ExpectQuery(`select name as db_name from sys\.databases`).
					WillReturnRows(databaseRows)

				mock.ExpectClose()
			},
			newDatabaseConnection: func() func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
				return getNewDatabaseConnection2
			},
			args: args.ArgumentList{
				EnableDatabaseReserveMetrics: true,
			},
			engineEdition: database.AzureSQLDatabaseEngineEditionNumber,
			expectedFile:  filepath.Join("..", "testdata", "partialAzureSQLDatabaseMetrics.json.golden"),
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runPopulateDatabaseMetricsTest(t, tc)
		})
	}
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
		attribute.Attribute{Key: "displayName", Value: "master"},
		attribute.Attribute{Key: "entityName", Value: "database:master"},
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

	// Enable logging if needed
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)

	tests := []struct {
		name               string
		engineEditionValue int
		perfCounterSetup   func(mock sqlmock.Sqlmock) // Function to mock performance counter query
		expectedFile       string                     // Path to the expected file
		args               args.ArgumentList          // Arguments to pass to PopulateInstanceMetrics
		updateDefinitions  func()
	}{
		{
			name:               "Engine edition 3: Collect all metrics",
			engineEditionValue: 3,
			perfCounterSetup: func(mock sqlmock.Sqlmock) {
				perfCounterRows := sqlmock.NewRows([]string{"buffer_cache_hit_ratio", "buffer_pool_hit_percent", "sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"}).
					AddRow(22, 100, 4736, 142, 3, 641, 2509, 848, 0, 67, 0, 18021, 1112946000, 184700, 0)
				mock.ExpectQuery(`SELECT\s+t1.cntr_value AS buffer_cache_hit_ratio.*`).WillReturnRows(perfCounterRows)
			},
			expectedFile: filepath.Join("..", "testdata", "perfCounter.json.golden"),
			args: args.ArgumentList{
				EnableBufferMetrics:      true,
				EnableDiskMetricsInBytes: true,
			},
		},
		{
			name:               "Engine edition 5: Skip unsupported metrics",
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber, // Azure SQL Database engine edition
			perfCounterSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT\s+physical_memory_in_use_kb\s+FROM\s+sys\.dm_os_process_memory`).WillReturnRows(sqlmock.NewRows([]string{"physical_memory_in_use_kb"}))
				mock.ExpectQuery(`SELECT\s+name\s+FROM\s+sys\.master_files`).WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			expectedFile: filepath.Join("..", "testdata", "empty.json.golden"),
			args: args.ArgumentList{
				EnableBufferMetrics:      true,
				EnableDiskMetricsInBytes: true,
			},
		},
		{
			name:               "Engine edition 5: Get non-empty metrics",
			engineEditionValue: database.AzureSQLDatabaseEngineEditionNumber,
			perfCounterSetup: func(mock sqlmock.Sqlmock) {
				// Mock the buffer_pool_hit_percent query
				perfCounterRows := sqlmock.NewRows([]string{"buffer_pool_hit_percent"}).
					AddRow(95.5)
				mock.ExpectQuery(`SELECT\s+\(a\.cntr_value\s+\*\s+1\.0\s+/\s+b\.cntr_value\)\s+\*\s+100\.0\s+AS\s+buffer_pool_hit_percent\s+FROM\s+sys\.dm_os_performance_counters\s+a\s+JOIN\s+\(SELECT\s+cntr_value,\s+OBJECT_NAME\s+FROM\s+sys\.dm_os_performance_counters\s+WHERE\s+counter_name\s+=\s+'Buffer cache hit ratio base'\)\s+b\s+ON\s+a\.OBJECT_NAME\s+=\s+b\.OBJECT_NAME\s+WHERE\s+a\.counter_name\s+=\s+'Buffer cache hit ratio'`).WillReturnRows(perfCounterRows)
			},
			expectedFile: filepath.Join("..", "testdata", "nonEmptyMetrics.json.golden"),
			args: args.ArgumentList{
				EnableBufferMetrics:      true,
				EnableDiskMetricsInBytes: true,
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, e := createTestEntity(t)

			conn, mock := connection.CreateMockSQL(t)
			defer conn.Close()

			tt.perfCounterSetup(mock)
			PopulateInstanceMetrics(e, conn, tt.args, tt.engineEditionValue)

			actual, _ := i.MarshalJSON()
			assert.NoError(t, updateGoldenFile(actual, tt.expectedFile))

			checkAgainstFile(t, actual, tt.expectedFile)
		})
	}
}

func Test_populateInstanceMetrics_NoReturn(t *testing.T) {
	i, e := createTestEntity(t)

	conn, mock := connection.CreateMockSQL(t)
	defer conn.Close()

	perfCounterRows := sqlmock.NewRows([]string{"buffer_cache_hit_ratio", "buffer_pool_hit_percent", "sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"})

	// only match the performance counter query
	mock.ExpectQuery(`SELECT\s+t1.cntr_value AS buffer_cache_hit_ratio.*`).WillReturnRows(perfCounterRows)
	mock.ExpectClose()

	args := args.ArgumentList{
		EnableBufferMetrics: true,
	}

	engineEdition := 3

	PopulateInstanceMetrics(e, conn, args, engineEdition)

	actual, _ := i.MarshalJSON()
	expectedFile := filepath.Join("..", "testdata", "empty.json.golden")
	assert.NoError(t, updateGoldenFile(actual, expectedFile))

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
	assert.NoError(t, updateGoldenFile(actual, expectedFile))

	checkAgainstFile(t, actual, expectedFile)
}

func Test_populateCustomQuery(t *testing.T) { //nolint: funlen
	cases := []struct {
		Name             string
		cq               customQuery
		setupMock        func(sqlmock.Sqlmock, customQuery)
		expectedFileName string
	}{
		{
			Name: "Custom metrics in query",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_name", "metric_value", "metric_type", "otherValue", "attrValue"}).
					AddRow("myMetric", 0.5, "gauge", 42, "aa").
					AddRow("myMetric", 1.5, "gauge", 43, "bb")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					'myMetric' as metric_name,
					value as metric_value,
					'gauge' as metric_type,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
			},
			expectedFileName: "customQuery.json",
		},
		{
			Name: "Custom metrics in config",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_value", "otherValue", "attrValue"}).
					AddRow(0.5, 42, "aa").
					AddRow(1.5, 43, "bb")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					value as metric_value,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
				Name:   "myMetric",
				Type:   "gauge",
				Prefix: "prefix_",
			},
			expectedFileName: "customQueryPrefix.json",
		},
		{
			Name: "Custom metrics in config, detecting type",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_value", "otherValue", "attrValue"}).
					AddRow(0.5, 42, "aa").
					AddRow(1.5, 43, "bb")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					value as metric_value,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
				Name:   "myMetric",
				Prefix: "prefix_",
			},
			expectedFileName: "customQueryPrefix.json",
		},
		{
			Name: "Custom metrics, query has precedence",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_name", "metric_value", "metric_type", "otherValue", "attrValue"}).
					AddRow("myMetric", 0.5, "gauge", 42, "aa").
					AddRow("myMetric", 1.5, "gauge", 43, "bb")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					'myMetric' as metric_name,
					value as metric_value,
					'gauge' as metric_type,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
				Name:   "other",
				Type:   "delta",
				Prefix: "prefix_",
			},
			expectedFileName: "customQueryPrefix.json",
		},
		{
			Name: "Custom metrics, query with null values",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_name", "metric_value", "metric_type", "otherValue", "attrValue"}).
					AddRow("myMetric", 0.5, "gauge", nil, nil).
					AddRow("myMetric", 1.5, "gauge", 43, nil).
					AddRow("myMetric", 2.5, "gauge", nil, "cc").
					AddRow("myMetric", nil, "gauge", 44, nil).
					AddRow("myMetric", 4.5, "gauge", 45, "dd")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					'myMetric' as metric_name,
					value as metric_value,
					'gauge' as metric_type,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
				Name:   "other",
				Type:   "delta",
				Prefix: "prefix_",
			},
			expectedFileName: "customQueryNull.json",
		},
		{
			Name: "Custom metrics in config with null values in query output",
			setupMock: func(mock sqlmock.Sqlmock, cq customQuery) {
				customQueryRows := sqlmock.NewRows([]string{"metric_value", "otherValue", "attrValue"}).
					AddRow(0.5, nil, nil).
					AddRow(1.5, 43, nil).
					AddRow(2.5, nil, "cc").
					AddRow(nil, 44, nil).
					AddRow(4.5, 45, "dd")
				mock.ExpectQuery(cq.Query).WillReturnRows(customQueryRows)
				mock.ExpectClose()
			},
			cq: customQuery{
				Query: `SELECT
					value as metric_value,
					value2 as 'otherValue'
					attr as 'attrValue'
					FROM my_table`,
				Name:   "myMetric",
				Type:   "gauge",
				Prefix: "prefix_",
			},
			expectedFileName: "customQueryNull.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { //nolint: paralleltest // setup mocks
			i, e := createTestEntity(t)
			conn, mock := connection.CreateMockSQL(t)
			defer conn.Close()
			tc.setupMock(mock, tc.cq)
			populateCustomMetrics(e, conn, tc.cq)
			actual, _ := i.MarshalJSON()
			expectedFile := filepath.Join("..", "testdata", tc.expectedFileName)
			checkAgainstFile(t, actual, expectedFile)
		})
	}
}

func Test_extractValue(t *testing.T) { //nolint: funlen
	cases := []struct {
		Name          string
		input         sql.NullString
		expectedValue string
	}{
		{
			Name:          "Valid NullString",
			input:         sql.NullString{String: "abc", Valid: true},
			expectedValue: "abc",
		},
		{
			Name:          "nil NullString",
			input:         sql.NullString{Valid: false},
			expectedValue: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) { //nolint: paralleltest // setup mocks
			actualValue := extractValue(tc.input)
			assert.Equal(t, tc.expectedValue, actualValue)
		})
	}
}
