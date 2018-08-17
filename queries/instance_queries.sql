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
t13.cntr_value as page_life_expectancy_sec
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
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13

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