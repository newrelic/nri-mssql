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
	errQueringDiskSpace     = errors.New("disk space error")
	errQueringUtilization   = errors.New("utilization error")
	errQueringMemory        = errors.New("total memory error")
	errQueringLogGrowth     = errors.New("log growth error")
	errQueringIOStalls      = errors.New("io stalls error")
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

type mockConnectionBuilder struct {
	mock   sqlmock.Sqlmock
	db     *sql.DB
	args   *args.ArgumentList
	dbName string
}

func newMockConnectionBuilder(args *args.ArgumentList, dbName string) (*mockConnectionBuilder, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	return &mockConnectionBuilder{
		mock:   mock,
		db:     db,
		args:   args,
		dbName: dbName,
	}, nil
}

// expectStandardQueries adds expectations for the standard log growth and IO stalls queries.
func (b *mockConnectionBuilder) expectStandardQueries() *mockConnectionBuilder {
	metricValue := 1
	if b.dbName == "db-1" {
		metricValue = 0
	}
	b.mock.ExpectQuery(`^SELECT\s+sd\.name\s+AS\s+db_name,\s+spc\.cntr_value\s+AS\s+log_growth*`).
		WillReturnRows(sqlmock.NewRows([]string{"db_name", "log_growth"}).AddRow(b.dbName, metricValue))
	b.mock.ExpectQuery(`^SELECT\s+DB_NAME\(\)\s+AS\s+db_name,\s+SUM\(io_stall\).*`).
		WillReturnRows(sqlmock.NewRows([]string{"db_name", "io_stalls"}).AddRow(b.dbName, metricValue))
	return b
}

// mockResponseType defines the possible outcomes for a mocked query.
type mockResponseType int

const (
	mockSuccess mockResponseType = iota // Return successful rows
	mockError                           // Return an error
	mockEmpty                           // Return an empty rowset
)

// expectMemoryAndDiskQueries adds expectations for memory and disk space, optionally returning errors.
func (b *mockConnectionBuilder) expectMemoryQueries(utilRespType mockResponseType, totalMemRespType mockResponseType) *mockConnectionBuilder {
	// Memory Utilization
	utilQuery := b.mock.ExpectQuery(`^SELECT\s+top\s+1\s+DB_NAME\(\)\s+AS\s+db_name,\s+avg_memory_usage_percent\s+AS\s+memory_utilization\s+FROM\s+sys\.dm_db_resource_stats\s+ORDER\s+BY\s+end_time\s+DESC;?$`)
	switch utilRespType {
	case mockError:
		utilQuery.WillReturnError(errQueringUtilization)
	case mockEmpty:
		utilQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "memory_utilization"}))
	case mockSuccess:
		utilQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "memory_utilization"}).AddRow(b.dbName, 31.18))
	default:
		utilQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "memory_utilization"}).AddRow(b.dbName, 31.18))
	}

	// Total Memory
	totalMemQuery := b.mock.ExpectQuery(`^SELECT\s+DB_NAME\(\)\s+AS\s+db_name,\s+\(process_memory_limit_mb\s+\*\s+1024\s+\*\s+1024\)\s+AS\s+total_physical_memory\s+FROM\s+sys\.dm_os_job_object;?$`)

	switch totalMemRespType {
	case mockError:
		totalMemQuery.WillReturnError(errQueringMemory)
	case mockEmpty:
		totalMemQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "total_physical_memory"}))
	case mockSuccess:
		totalMemQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "total_physical_memory"}).AddRow(b.dbName, 2097152))
	default:
		totalMemQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "total_physical_memory"}).AddRow(b.dbName, 2097152))
	}
	return b
}

// expectBufferQueries adds expectations for buffer metrics if enabled.
func (b *mockConnectionBuilder) expectDiskQueries(mockRespType mockResponseType) *mockConnectionBuilder {
	if b.args.EnableDiskMetricsInBytes {
		// Disk Space
		diskQuery := b.mock.ExpectQuery(`^SELECT\s+DB_NAME\(\)\s+AS\s+db_name,\s+CAST\(DATABASEPROPERTYEX\(DB_NAME\(\),\s+'MaxSizeInBytes'\)\s+AS\s+BIGINT\)\s+AS\s+max_disk_space;?$`)
		switch mockRespType {
		case mockError:
			diskQuery.WillReturnError(errQueringDiskSpace)
		case mockEmpty:
			diskQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "max_disk_space"}))
		case mockSuccess:
			diskQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "max_disk_space"}).AddRow(b.dbName, 104857600))
		default:
			diskQuery.WillReturnRows(sqlmock.NewRows([]string{"db_name", "max_disk_space"}).AddRow(b.dbName, 104857600))
		}
	}
	return b
}

// expectBufferQueries adds expectations for buffer metrics if enabled.
func (b *mockConnectionBuilder) expectBufferQueries() *mockConnectionBuilder {
	if b.args.EnableBufferMetrics {
		metricValue := 1
		if b.dbName == "db-1" {
			metricValue = 0
		}
		b.mock.ExpectQuery(`^SELECT\s+DB_NAME\(\)\s+AS\s+db_name.*buffer_pool_size.*`).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "buffer_pool_size"}).AddRow(b.dbName, metricValue))
	}
	return b
}

// expectReserveQueries adds expectations for reserve space metrics if enabled.
func (b *mockConnectionBuilder) expectReserveQueries() *mockConnectionBuilder {
	if b.args.EnableDatabaseReserveMetrics {
		metricValue := 1
		if b.dbName == "db-1" {
			metricValue = 0
		}
		b.mock.ExpectQuery(`^SELECT\s+DB_NAME\(\)\s+AS\s+db_name.*reserved_space.*`).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "reserved_space", "reserved_space_not_used"}).AddRow(b.dbName, metricValue, metricValue))
	}
	return b
}

func (b *mockConnectionBuilder) build() (*connection.SQLConnection, error) {
	b.mock.ExpectClose()
	return &connection.SQLConnection{Connection: sqlx.NewDb(b.db, "sqlmock"), Host: "test_host"}, nil
}

type DatabaseMetricsTesCase struct {
	name                  string
	setupMock             func(sqlmock.Sqlmock)
	newDatabaseConnection func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error)
	args                  args.ArgumentList
	engineEdition         int
	expectedFile          string
	expectError           bool
}

func runPopulateDatabaseMetricsTest(
	t *testing.T,
	tc DatabaseMetricsTesCase,
) {
	i, _ := createTestEntity(t)
	conn, mock := connection.CreateMockSQL(t)
	tc.setupMock(mock)
	// Override NewDatabaseConnection
	originalNewDatabaseConnection := connection.CreateDatabaseConnection
	defer func() {
		connection.CreateDatabaseConnection = originalNewDatabaseConnection
	}()

	connection.CreateDatabaseConnection = tc.newDatabaseConnection

	assert.NoError(t, PopulateDatabaseMetrics(i, "MSSQL", conn, tc.args, tc.engineEdition))

	actual, _ := i.MarshalJSON()
	assert.NoError(t, updateGoldenFile(actual, tc.expectedFile))
	checkAgainstFile(t, actual, tc.expectedFile)
}

func setupMockForDatabaseMetrics(mock sqlmock.Sqlmock, logGrowthResp mockResponseType, ioStallsResp mockResponseType, args args.ArgumentList, engineEdition int) {
	databaseRows := sqlmock.NewRows([]string{"db_name"}).AddRow("db-1").AddRow("db-2")
	mock.ExpectQuery(`select name as db_name from sys\.databases`).WillReturnRows(databaseRows)

	var logGrowthRegex string
	var ioStallsRegex string

	if engineEdition == database.AzureSQLManagedInstanceEngineEditionNumber {
		logGrowthRegex = `^SELECT\s+sd\.name\s+AS\s+db_name,\s+spc\.cntr_value\s+AS\s+log_growth.*`
		ioStallsRegex = `^SELECT\s+DB_NAME\(database_id\)\s+AS\s+db_name,\s+SUM\(io_stall\)\s+AS\s+io_stalls.*`
	} else {
		logGrowthRegex = `^select\s+RTRIM\(t1\.instance_name\).*`
		ioStallsRegex = `^select.*as\s+io_stalls.*FROM\s+sys\.dm_io_virtual_file_stats.*`
	}

	switch logGrowthResp {
	case mockSuccess:
		mock.ExpectQuery(logGrowthRegex).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "log_growth"}).AddRow("db-1", 0).AddRow("db-2", 1))
	case mockError:
		mock.ExpectQuery(logGrowthRegex).WillReturnError(errQueringLogGrowth)
	case mockEmpty:
		mock.ExpectQuery(logGrowthRegex).WillReturnRows(sqlmock.NewRows([]string{"db_name", "log_growth"}))
	}

	switch ioStallsResp {
	case mockSuccess:
		mock.ExpectQuery(ioStallsRegex).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "io_stalls"}).AddRow("db-1", 0).AddRow("db-2", 1))
	case mockError:
		mock.ExpectQuery(ioStallsRegex).WillReturnError(errQueringIOStalls)
	case mockEmpty:
		mock.ExpectQuery(ioStallsRegex).WillReturnRows(sqlmock.NewRows([]string{"db_name", "io_stalls"}))
	}

	if args.EnableBufferMetrics {
		bufferRegex := `^SELECT DB_NAME\(database_id\) AS db_name, buffer_pool_size \* \(8\*1024\) AS buffer_pool_size .*`
		mock.ExpectQuery(bufferRegex).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "buffer_pool_size"}).AddRow("db-1", 0).AddRow("db-2", 1))
	}

	if args.EnableDatabaseReserveMetrics {
		reserveRegex := `^USE\s+"[^"]+"\s+;\s*WITH\s+reserved_space.*`
		mock.ExpectQuery(reserveRegex).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "reserved_space", "reserved_space_not_used"}).AddRow("db-1", 0, 0))
		mock.ExpectQuery(reserveRegex).
			WillReturnRows(sqlmock.NewRows([]string{"db_name", "reserved_space", "reserved_space_not_used"}).AddRow("db-2", 1, 1))
	}
}

func Test_populateDatabaseMetrics(t *testing.T) {
	// Enable logging if needed
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)

	scenarios := []struct {
		name          string
		logGrowthResp mockResponseType
		ioStallsResp  mockResponseType
		args          args.ArgumentList
		expectedFile  string
	}{
		{
			name:          "Collect all metrics",
			logGrowthResp: mockSuccess,
			ioStallsResp:  mockSuccess,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: true,
			},
			expectedFile: "databaseMetrics.json.golden",
		},
		{
			name:          "Error querying log_growth, io_stalls metrics",
			logGrowthResp: mockError,
			ioStallsResp:  mockError,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: false,
			},
			expectedFile: "databaseMetricsWithoutDefaultDBMetrics.json.golden",
		},
		{
			name:          "Empty output from log_growth and io_stalls queries",
			logGrowthResp: mockEmpty,
			ioStallsResp:  mockEmpty,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: false,
			},
			expectedFile: "databaseMetricsWithoutDefaultDBMetrics.json.golden",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			tc := DatabaseMetricsTesCase{
				setupMock: func(s sqlmock.Sqlmock) {
					setupMockForDatabaseMetrics(s, sc.logGrowthResp, sc.ioStallsResp, sc.args, 3)
				},
				newDatabaseConnection: func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
					return nil, nil
				},
				args:          sc.args,
				engineEdition: 3,
				expectedFile:  filepath.Join("..", "testdata", sc.expectedFile),
			}
			runPopulateDatabaseMetricsTest(t, tc)
		})
	}
}

func setupMockExpectedMetricsForAzureSQLDatabase(mock sqlmock.Sqlmock) {
	databaseRows := sqlmock.NewRows([]string{"db_name"}).
		AddRow("db-1").
		AddRow("db-2")

	mock.ExpectQuery(`select name as db_name from sys\.databases`).
		WillReturnRows(databaseRows)

	mock.ExpectClose()
}

func Test_populateDatabaseMetrics_AzureSQLDatabase(t *testing.T) {
	// Enable logging if needed
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)

	scenarios := []struct {
		name         string
		dbNameToFail string
		memUtilResp  mockResponseType
		totalMemResp mockResponseType
		diskResp     mockResponseType
		args         args.ArgumentList
		expectedFile string
	}{
		{
			name:         "Collect all metrics",
			memUtilResp:  mockSuccess,
			totalMemResp: mockSuccess,
			diskResp:     mockSuccess,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: true,
				EnableDiskMetricsInBytes:     true,
			},
			expectedFile: "azureSQLDatabaseMetrics.json.golden",
		},
		{
			name:         "Error creating connection to db-1",
			dbNameToFail: "db-1",
			args: args.ArgumentList{
				EnableDatabaseReserveMetrics: true,
				EnableDiskMetricsInBytes:     true,
			},
			expectedFile: "partialAzureSQLDatabaseMetrics.json.golden",
		},
		{
			name:         "Error collecting memory and disk metrics",
			memUtilResp:  mockError,
			totalMemResp: mockError,
			diskResp:     mockError,
			args: args.ArgumentList{
				EnableDatabaseReserveMetrics: true,
				EnableDiskMetricsInBytes:     true,
			},
			expectedFile: "azureSQLDatabaseMetricsWithoutMemoryMetrics.json.golden",
		},
		{
			name:         "Collecting partial memory metrics and disabling disk metrics",
			memUtilResp:  mockSuccess,
			totalMemResp: mockError,
			diskResp:     mockSuccess,
			args: args.ArgumentList{
				EnableDatabaseReserveMetrics: true,
				EnableDiskMetricsInBytes:     false,
			},
			expectedFile: "databasePartialMemoryMetrics.json.golden",
		},
		{
			name:         "Empty output from memory and disk queries",
			memUtilResp:  mockEmpty,
			totalMemResp: mockEmpty,
			diskResp:     mockEmpty,
			args: args.ArgumentList{
				EnableDatabaseReserveMetrics: true,
				EnableDiskMetricsInBytes:     true,
			},
			expectedFile: "azureSQLDatabaseMetricsWithoutMemoryMetrics.json.golden",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			tc := DatabaseMetricsTesCase{
				setupMock: setupMockExpectedMetricsForAzureSQLDatabase,
				newDatabaseConnection: func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
					if dbName == sc.dbNameToFail {
						return nil, errCreateConnectionToDB
					}
					builder, err := newMockConnectionBuilder(args, dbName)
					if err != nil {
						return nil, err
					}
					return builder.
						expectStandardQueries().
						expectMemoryQueries(sc.memUtilResp, sc.totalMemResp).
						expectDiskQueries(sc.diskResp).
						expectBufferQueries().
						expectReserveQueries().
						build()
				},
				args:          sc.args,
				engineEdition: database.AzureSQLDatabaseEngineEditionNumber,
				expectedFile:  filepath.Join("..", "testdata", sc.expectedFile),
			}
			runPopulateDatabaseMetricsTest(t, tc)
		})
	}
}

func Test_populateDatabaseMetrics_AzureSQLManagedInstance(t *testing.T) {
	// Enable logging if needed
	// log.SetupLogging(true)
	// defer log.SetupLogging(false)

	scenarios := []struct {
		name          string
		logGrowthResp mockResponseType
		ioStallsResp  mockResponseType
		args          args.ArgumentList
		expectedFile  string
	}{
		{
			name:          "Collect all metrics",
			logGrowthResp: mockSuccess,
			ioStallsResp:  mockSuccess,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: true,
			},
			expectedFile: "azureSQLManagedInstanceMetrics.json.golden",
		},
		{
			name:          "Error querying log_growth, io_stalls metrics",
			logGrowthResp: mockError,
			ioStallsResp:  mockError,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: false,
			},
			expectedFile: "azureSQLManagedInstanceMetricsWithoutDefaultDBMetrics.json.golden",
		},
		{
			name:          "Empty output from log_growth and io_stalls queries",
			logGrowthResp: mockEmpty,
			ioStallsResp:  mockEmpty,
			args: args.ArgumentList{
				EnableBufferMetrics:          true,
				EnableDatabaseReserveMetrics: false,
			},
			expectedFile: "azureSQLManagedInstanceMetricsWithoutDefaultDBMetrics.json.golden",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			tc := DatabaseMetricsTesCase{
				setupMock: func(s sqlmock.Sqlmock) {
					setupMockForDatabaseMetrics(s, sc.logGrowthResp, sc.ioStallsResp, sc.args, database.AzureSQLManagedInstanceEngineEditionNumber)
				},
				newDatabaseConnection: func(args *args.ArgumentList, dbName string) (*connection.SQLConnection, error) {
					return nil, nil
				},
				args:          sc.args,
				engineEdition: database.AzureSQLManagedInstanceEngineEditionNumber,
				expectedFile:  filepath.Join("..", "testdata", sc.expectedFile),
			}
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
				perfCounterRows := sqlmock.NewRows([]string{"sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"}).
					AddRow(4736, 142, 3, 641, 2509, 848, 0, 67, 0, 18021, 1112946000, 184700, 0)
				mock.ExpectQuery(`^SELECT.*t1.cntr_value AS sql_compilations*`).WillReturnRows(perfCounterRows)
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
		},
		{
			name:               "Engine edition 8: Get instance memory metrics",
			engineEditionValue: database.AzureSQLManagedInstanceEngineEditionNumber,
			perfCounterSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`^SELECT.*AS\s+memory_utilization\s+FROM.*WHERE\s+object_name\s+LIKE\s+'%:Memory Manager%'`).
					WillReturnRows(sqlmock.NewRows([]string{"total_physical_memory", "available_physical_memory", "memory_utilization"}).
						AddRow(100, 20, 80))
			},
			expectedFile: filepath.Join("..", "testdata", "instanceMemoryMetrics.json.golden"),
			args: args.ArgumentList{
				EnableBufferMetrics:      true,
				EnableDiskMetricsInBytes: true,
			},
		},
	}

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

	perfCounterRows := sqlmock.NewRows([]string{"sql_compilations", "sql_recompilations", "user_connections", "lock_wait_time_ms", "page_splits_sec", "checkpoint_pages_sec", "deadlocks_sec", "user_errors", "kill_connection_errors", "batch_request_sec", "page_life_expectancy_ms", "transactions_sec", "forced_parameterizations_sec"})

	// only match the performance counter query
	mock.ExpectQuery(`SELECT\s+t1.cntr_value AS sql_compilations.*`).WillReturnRows(perfCounterRows)
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
