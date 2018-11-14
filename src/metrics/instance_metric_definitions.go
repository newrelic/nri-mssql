package metrics

var instanceDefinitions = []*QueryDefinition{
	{
		query: `SELECT
		t1.cntr_value AS buffer_cache_hit_ratio,
		(t1.cntr_value * 1.0 / t2.cntr_value) * 100.0 AS buffer_pool_hit_percent,
		t3.cntr_value AS sql_compilations,
		t4.cntr_value AS sql_recompilations,
		t5.cntr_value AS user_connections,
		t6.cntr_value AS lock_wait_time_ms,
		t7.cntr_value AS page_splits_sec,
		t8.cntr_value AS checkpoint_pages_sec,
		t9.cntr_value AS deadlocks_sec,
		t10.cntr_value AS user_errors,
		t11.cntr_value AS kill_connection_errors,
		t12.cntr_value AS batch_request_sec,
		(t13.cntr_value * 1000.0) AS page_life_expectancy_ms,
		t14.cntr_value AS transactions_sec,
		t15.cntr_value AS forced_parameterizations_sec
		FROM (SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Buffer cache hit ratio') t1,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Buffer cache hit ratio base') t2,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'SQL Compilations/sec') t3,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'SQL Re-Compilations/sec') t4,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'User Connections') t5,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Lock Wait Time (ms)' AND instance_name = '_Total') t6,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Page Splits/sec') t7,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Checkpoint pages/sec') t8,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total') t9,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE object_name LIKE '%SQL Errors%' AND instance_name = 'User Errors') t10,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE object_name LIKE '%SQL Errors%' AND instance_name LIKE 'Kill Connection Errors%') t11,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Batch Requests/sec') t12,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13,
		(SELECT Sum(cntr_value) AS cntr_value FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Transactions/sec') t14,
		(SELECT * FROM sys.dm_os_performance_counters WITH (nolock) WHERE counter_name = 'Forced Parameterizations/sec') t15`,
		dataModels: &[]struct {
			BufferCacheHitRatio        *int     `db:"buffer_cache_hit_ratio" metric_name:"buffer.cacheHitRatio" source_type:"gauge"`
			BufferPoolHitPercent       *float64 `db:"buffer_pool_hit_percent" metric_name:"system.bufferPoolHit" source_type:"gauge"`
			SQLCompilations            *int     `db:"sql_compilations" metric_name:"stats.sqlCompilationsPerSecond" source_type:"rate"`
			SQLRecompilations          *int     `db:"sql_recompilations" metric_name:"stats.sqlRecompilationsPerSecond" source_type:"rate"`
			UserConnections            *int     `db:"user_connections" metric_name:"stats.connections" source_type:"gauge"`
			LockWaitTimeMs             *int     `db:"lock_wait_time_ms" metric_name:"stats.lockWaitsPerSecond" source_type:"rate"`
			PageSplitsSec              *int     `db:"page_splits_sec" metric_name:"access.pageSplitsPerSecond" source_type:"rate"`
			CheckpointPagesSec         *int     `db:"checkpoint_pages_sec" metric_name:"buffer.checkpointPagesPerSecond" source_type:"rate"`
			DeadlocksSec               *int     `db:"deadlocks_sec" metric_name:"stats.deadlocksPerSecond" source_type:"rate"`
			UserErrors                 *int     `db:"user_errors" metric_name:"stats.userErrorsPerSecond" source_type:"rate"`
			KillConnectionErrors       *int     `db:"kill_connection_errors" metric_name:"stats.killConnectionErrorsPerSecond" source_type:"rate"`
			BatchRequestSec            *int     `db:"batch_request_sec" metric_name:"bufferpool.batchRequestsPerSecond" source_type:"rate"`
			PageLifeExpectancySec      *float64 `db:"page_life_expectancy_ms" metric_name:"bufferpool.pageLifeExpectancyInMilliseconds" source_type:"gauge"`
			TransactionsSec            *int     `db:"transactions_sec" metric_name:"instance.transactionsPerSecond" source_type:"rate"`
			ForcedParameterizationsSec *int     `db:"forced_parameterizations_sec" metric_name:"instance.forcedParameterizationsPerSecond" source_type:"rate"`
		}{},
	},
	{
		query: `SELECT
		Sum(wait_time_ms) AS wait_time
		FROM sys.dm_os_wait_stats
		WHERE [wait_type] NOT IN (
		N'CLR_SEMAPHORE',    N'LAZYWRITER_SLEEP',
		N'RESOURCE_QUEUE',   N'SQLTRACE_BUFFER_FLUSH',
		N'SLEEP_TASK',       N'SLEEP_SYSTEMTASK',
		N'WAITFOR',          N'HADR_FILESTREAM_IOMGR_IOCOMPLETION',
		N'CHECKPOINT_QUEUE', N'REQUEST_FOR_DEADLOCK_SEARCH',
		N'XE_TIMER_EVENT',   N'XE_DISPATCHER_JOIN',
		N'LOGMGR_QUEUE',     N'FT_IFTS_SCHEDULER_IDLE_WAIT',
		N'BROKER_TASK_STOP', N'CLR_MANUAL_EVENT',
		N'CLR_AUTO_EVENT',   N'DISPATCHER_QUEUE_SEMAPHORE',
		N'TRACEWRITE',       N'XE_DISPATCHER_WAIT',
		N'BROKER_TO_FLUSH',  N'BROKER_EVENTHANDLER',
		N'FT_IFTSHC_MUTEX',  N'SQLTRACE_INCREMENTAL_FLUSH_SLEEP',
		N'DIRTY_PAGE_POLL',  N'SP_SERVER_DIAGNOSTICS_SLEEP')`,
		dataModels: &[]struct {
			WaitTime *int `db:"wait_time" metric_name:"system.waitTimeInMillisecondsPerSecond" source_type:"rate"`
		}{},
	},
	{
		query: `SELECT
		Max(CASE WHEN sessions.status = 'preconnect' THEN counts ELSE 0 END) AS preconnect,
		Max(CASE WHEN sessions.status = 'background' THEN counts ELSE 0 END) AS background,
		Max(CASE WHEN sessions.status = 'dormant' THEN counts ELSE 0 END) AS dormant,
		Max(CASE WHEN sessions.status = 'runnable' THEN counts ELSE 0 END) AS runnable,
		Max(CASE WHEN sessions.status = 'suspended' THEN counts ELSE 0 END) AS suspended,
		Max(CASE WHEN sessions.status = 'running' THEN counts ELSE 0 END) AS running,
		Max(CASE WHEN sessions.status = 'blocked' THEN counts ELSE 0 END) AS blocked,
		Max(CASE WHEN sessions.status = 'sleeping' THEN counts ELSE 0 END) AS sleeping
		FROM (SELECT status, Count(*) counts FROM (
			SELECT CASE WHEN req.status IS NOT NULL THEN
				CASE WHEN req.blocking_session_id <> 0 THEN 'blocked' ELSE req.status END
			  ELSE sess.status END status, req.blocking_session_id
			FROM sys.dm_exec_sessions sess
			LEFT JOIN sys.dm_exec_requests req
			ON sess.session_id = req.session_id
			WHERE sess.session_id > 50 ) statuses
		  GROUP BY status) sessions`,
		dataModels: &[]struct {
			Preconnect *int `db:"preconnect" metric_name:"instance.preconnectProcessesCount" source_type:"gauge"`
			Background *int `db:"background" metric_name:"instance.backgroundProcessesCount" source_type:"gauge"`
			Dormant    *int `db:"dormant" metric_name:"instance.dormantProcessesCount" source_type:"gauge"`
			Runnable   *int `db:"runnable" metric_name:"instance.runnableProcessesCount" source_type:"gauge"`
			Suspended  *int `db:"suspended" metric_name:"instance.suspendedProcessesCount" source_type:"gauge"`
			Running    *int `db:"running" metric_name:"instance.runningProcessesCount" source_type:"gauge"`
			Blocked    *int `db:"blocked" metric_name:"instance.blockedProcessesCount" source_type:"gauge"`
			Sleeping   *int `db:"sleeping" metric_name:"instance.sleepingProcessesCount" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT Sum(total_bytes) AS total_disk_space FROM (
			SELECT DISTINCT
			dovs.volume_mount_point,
			dovs.available_bytes available_bytes,
			dovs.total_bytes total_bytes
			FROM sys.master_files mf
			CROSS apply sys.Dm_os_volume_stats(mf.database_id, mf.file_id) dovs
			) drives`,
		dataModels: &[]struct {
			TotalDiskSpace *int `db:"total_disk_space" metric_name:"instance.diskInBytes" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT Sum(runnable_tasks_count) AS runnable_tasks_count
		FROM sys.dm_os_schedulers
		WHERE   scheduler_id < 255 AND [status] = 'VISIBLE ONLINE'`,
		dataModels: &[]struct {
			RunnableTasksCount *int `db:"runnable_tasks_count" metric_name:"instance.runnableTasks" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT Sum(db.buffer_pool_size) AS instance_buffer_pool_size FROM (
			SELECT
			Count_big(*) * (8*1024) AS buffer_pool_size
			FROM sys.dm_os_buffer_descriptors WITH (nolock)
			WHERE database_id <> 32767 -- ResourceDB
			GROUP BY database_id) db`,
		dataModels: &[]struct {
			InstanceBufferPoolSize *int `db:"instance_buffer_pool_size" metric_name:"bufferpool.sizeInBytes" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT Sum(db.active_connections) AS instance_active_connections FROM (
			SELECT
			Count(syssp.dbid) AS active_connections
			FROM sys.databases db WITH (nolock)
			LEFT JOIN sys.sysprocesses syssp WITH (nolock) ON syssp.dbid = db.database_id
			GROUP BY db.NAME) db`,
		dataModels: &[]struct {
			InstanceActiveConnections *int `db:"instance_active_connections" metric_name:"activeConnections" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT
		Max(sys_mem.total_physical_memory_kb * 1024.0) AS total_physical_memory,
		Max(sys_mem.available_physical_memory_kb * 1024.0) AS available_physical_memory,
		(Max(proc_mem.physical_memory_in_use_kb) / (Max(sys_mem.total_physical_memory_kb) * 1.0)) * 100 AS memory_utilization
		FROM sys.dm_os_process_memory proc_mem,
		  sys.dm_os_sys_memory sys_mem,
		  sys.dm_os_performance_counters perf_count WHERE object_name = 'SQLServer:Memory Manager'`,
		dataModels: &[]struct {
			TotalPhysicalMemory     *float64 `db:"total_physical_memory" metric_name:"memoryTotal" source_type:"gauge"`
			AvailablePhysicalMemory *float64 `db:"available_physical_memory" metric_name:"memoryAvailable" source_type:"gauge"`
			MemoryUtilization       *float64 `db:"memory_utilization" metric_name:"memoryUtilization" source_type:"gauge"`
		}{},
	},
}

var waitTimeQuery = `SELECT wait_type, wait_time_ms AS wait_time, waiting_tasks_count
FROM sys.dm_os_wait_stats wait_stats
WHERE wait_time_ms != 0`

type waitTimeModel struct {
	WaitType  *string `db:"wait_type"`
	WaitTime  *int    `db:"wait_time"`
	WaitCount *int    `db:"waiting_tasks_count"`
}
