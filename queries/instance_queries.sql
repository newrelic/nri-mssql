select 
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
t13.cntr_value as page_life_expectancy_sec,
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
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name LIKE '%SQL Errors%' and instance_name = 'User Errors') t10,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name LIKE '%SQL Errors%' and instance_name like 'Kill Connection Errors%') t11,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Batch Requests/sec') t12,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13,
(SELECT SUM(cntr_value) as cntr_value FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Transactions/sec') t14,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Forced Parameterizations/sec') t15

SELECT
SUM(wait_time_ms) as wait_time
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
N'DIRTY_PAGE_POLL',  N'SP_SERVER_DIAGNOSTICS_SLEEP')

SELECT wait_type, wait_time_ms AS wait_time, waiting_tasks_count
      FROM sys.dm_os_wait_stats wait_stats
      WHERE wait_time_ms != 0

SELECT
MAX(CASE WHEN sessions.status = 'preconnect' then counts else 0 end) AS preconnect,
MAX(CASE WHEN sessions.status = 'background' then counts else 0 end) AS background,
MAX(CASE WHEN sessions.status = 'dormant' then counts else 0 end) AS dormant,
MAX(CASE WHEN sessions.status = 'runnable' then counts else 0 end) AS runnable,
MAX(CASE WHEN sessions.status = 'suspended' then counts else 0 end) AS suspended,
MAX(CASE WHEN sessions.status = 'running' then counts else 0 end) AS running,
MAX(CASE WHEN sessions.status = 'blocked' then counts else 0 end) AS blocked,
MAX(CASE WHEN sessions.status = 'sleeping' then counts else 0 end) AS sleeping
FROM (SELECT status, count(*) counts FROM (
        SELECT CASE WHEN req.status IS NOT NULL THEN
                CASE WHEN req.blocking_session_id <> 0 THEN 'blocked' ELSE req.status END
            ELSE sess.status END status, req.blocking_session_id
        FROM sys.dm_exec_sessions sess
        LEFT JOIN sys.dm_exec_requests req
        on sess.session_id = req.session_id
        WHERE sess.session_id > 50 ) statuses
    GROUP BY status) sessions

select Sum(total_bytes) AS total_disk_space from (
      select DISTINCT
      dovs.volume_mount_point,
      dovs.available_bytes available_bytes,
      dovs.total_bytes total_bytes
      FROM sys.master_files mf
      CROSS APPLY sys.dm_os_volume_stats(mf.database_id, mf.FILE_ID) dovs
      ) drives

select SUM(runnable_tasks_count) AS runnable_tasks_count
from sys.dm_os_schedulers
WHERE   scheduler_id < 255 AND [status] = 'VISIBLE ONLINE'

SELECT SUM(db.buffer_pool_size) as instance_buffer_pool_size from (
SELECT
COUNT_BIG(*) * (8*1024) AS buffer_pool_size
FROM sys.dm_os_buffer_descriptors WITH (NOLOCK)
WHERE database_id <> 32767 -- ResourceDB
GROUP BY database_id) db

SELECT SUM(db.active_connections) as instance_active_connections from (
SELECT
COUNT(syssp.dbid) AS active_connections
FROM sys.databases db WITH (NOLOCK)
LEFT JOIN sys.sysprocesses syssp WITH (NOLOCK) ON syssp.dbid = db.database_id
GROUP BY db.name) db

select
MAX(sys_mem.total_physical_memory_kb * 1024.0) AS total_physical_memory,
MAX(sys_mem.available_physical_memory_kb * 1024.0) AS available_physical_memory,
(Max(proc_mem.physical_memory_in_use_kb) / (Max(sys_mem.total_physical_memory_kb) * 1.0)) * 100 as memory_utilization
FROM sys.dm_os_process_memory proc_mem,
    sys.dm_os_sys_memory sys_mem,
    sys.dm_os_performance_counters perf_count WHERE object_name = 'SQLServer:Memory Manager'

select COALESCE( @@SERVERNAME, SERVERPROPERTY('ServerName'), SERVERPROPERTY('MachineName')) as instance_name
