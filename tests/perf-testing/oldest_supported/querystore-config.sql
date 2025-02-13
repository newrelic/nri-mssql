USE [master];
GO

RESTORE DATABASE [AdventureWorks2017]
FROM DISK = '/var/opt/mssql/backup/adventureworks-light.bak'
WITH 
    MOVE 'AdventureWorksLT2012_Data' TO '/var/opt/mssql/data/AdventureWorksLT2017.mdf',
    MOVE 'AdventureWorksLT2012_Log' TO '/var/opt/mssql/data/AdventureWorksLT2017_log.ldf',
    FILE = 1,
    NOUNLOAD,
    STATS = 5;
GO

-- Now enable Query Store
USE [AdventureWorks2017];
GO

ALTER DATABASE AdventureWorks2017 SET QUERY_STORE = ON
(
    OPERATION_MODE = READ_WRITE,
    CLEANUP_POLICY = (STALE_QUERY_THRESHOLD_DAYS = 30),
    DATA_FLUSH_INTERVAL_SECONDS = 900,
    MAX_STORAGE_SIZE_MB = 1000,
    INTERVAL_LENGTH_MINUTES = 60,
    SIZE_BASED_CLEANUP_MODE = AUTO,
    MAX_PLANS_PER_QUERY = 200,
    WAIT_STATS_CAPTURE_MODE = ON,
    QUERY_CAPTURE_MODE = ALL
);
GO