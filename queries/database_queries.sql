select name as db_name from sys.databases

select 
RTRIM(t1.instance_name) as db_name,
t1.cntr_value as log_growth
from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE object_name = 'SQLServer:Databases' and counter_name = 'Log Growths' and instance_name NOT IN ('_Total', 'mssqlsystemresource')) t1

select 
DB_NAME(database_id) AS db_name,
SUM(io_stall_write_ms) + SUM(num_of_writes) as io_stalls
FROM sys.dm_io_virtual_file_stats(null,null)
GROUP BY database_id

/*replace with database name*/
USE "master"
;WITH reserved_space(db_name, reserved_space_kb, reserved_space_not_used_kb)
AS
(
SELECT
    DB_NAME() AS db_name,
    sum(a.total_pages)*8.0 reserved_space_kb,
    sum(a.total_pages)*8.0 -sum(a.used_pages)*8.0 reserved_space_not_used_kb
FROM sys.partitions p
INNER JOIN sys.allocation_units a ON p.partition_id = a.container_id
LEFT JOIN sys.internal_tables it ON p.object_id = it.object_id
)
SELECT
db_name as db_name,
max(reserved_space_kb) * 1024 AS reserved_space,
max(reserved_space_not_used_kb) * 1024 AS reserved_space_not_used
FROM reserved_space
GROUP BY db_name

SELECT
DB_NAME(database_id) AS db_name,
COUNT_BIG(*) * (8*1024) AS buffer_pool_size
FROM sys.dm_os_buffer_descriptors WITH (NOLOCK)
WHERE database_id <> 32767 -- ResourceDB
GROUP BY database_id