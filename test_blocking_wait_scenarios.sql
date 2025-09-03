-- ========================================
-- UPDATED Test Scenarios for Blocking and Wait Analysis
-- ========================================

-- Use your test database (replace with your actual database name)
USE [AdventureWorks2022]; -- or your test database
GO

-- ========================================
-- SCENARIO 1: SIMPLE BLOCKING TEST
-- ========================================

-- Session 1: Run this first (creates a lock)
BEGIN TRANSACTION;
UPDATE Person.Person SET FirstName = FirstName WHERE BusinessEntityID = 1;
-- Don't commit yet - this will hold the lock

-- Session 2: Run this in a different connection (will be blocked)
SELECT FirstName FROM Person.Person WHERE BusinessEntityID = 1;

-- Session 3: Test the UPDATED blocking query (should now work)
DECLARE @Limit INT = 20;
DECLARE @TextTruncateLimit INT = 4094;

-- Simplified blocking detection
WITH DirectBlocking AS (
    SELECT
        req.blocking_session_id AS blocking_spid,
        req.session_id AS blocked_spid,
        req.wait_type AS wait_type,
        req.wait_time / 1000.0 AS wait_time_in_seconds,
        req.start_time AS start_time,
        sess.status AS blocked_status,
        req.command AS command_type,
        req.database_id AS database_id,
        req.sql_handle AS blocked_sql_handle,
        blocking_req.sql_handle AS blocking_sql_handle,
        blocking_req.start_time AS blocking_start_time,
        blocking_sess.status AS blocking_status,
        req.total_elapsed_time / 1000.0 AS total_elapsed_time_seconds,
        req.cpu_time / 1000.0 AS cpu_time_seconds,
        req.logical_reads,
        req.writes
    FROM
        sys.dm_exec_requests AS req WITH (NOLOCK)
    LEFT JOIN sys.dm_exec_requests AS blocking_req WITH (NOLOCK) 
        ON blocking_req.session_id = req.blocking_session_id
    LEFT JOIN sys.dm_exec_sessions AS sess WITH (NOLOCK) 
        ON sess.session_id = req.session_id
    LEFT JOIN sys.dm_exec_sessions AS blocking_sess WITH (NOLOCK) 
        ON blocking_sess.session_id = req.blocking_session_id
    WHERE
        req.blocking_session_id > 0
        AND req.session_id != req.blocking_session_id
        AND req.database_id > 4
        AND req.wait_time > 0  -- Any wait time for testing
)
SELECT TOP (@Limit)
    db.blocking_spid,
    db.blocking_status,
    db.blocked_spid,
    db.blocked_status,
    db.wait_type,
    db.wait_time_in_seconds,
    db.command_type,
    db.start_time AS blocked_query_start_time,
    DB_NAME(db.database_id) AS database_name,
    CASE
        WHEN blocking_sql.text IS NOT NULL THEN 
            LEFT(blocking_sql.text, @TextTruncateLimit)
        WHEN input_buffer.event_info IS NOT NULL THEN 
            LEFT(input_buffer.event_info, @TextTruncateLimit)
        ELSE 'No blocking query text available'
    END AS blocking_query_text,
    LEFT(COALESCE(blocked_sql.text, 'No blocked query text available'), @TextTruncateLimit) AS blocked_query_text,
    db.total_elapsed_time_seconds,
    db.cpu_time_seconds,
    db.logical_reads,
    db.writes,
    SYSDATETIME() AS collection_timestamp
FROM
    DirectBlocking db
OUTER APPLY sys.dm_exec_sql_text(db.blocking_sql_handle) AS blocking_sql
OUTER APPLY sys.dm_exec_sql_text(db.blocked_sql_handle) AS blocked_sql
OUTER APPLY sys.dm_exec_input_buffer(db.blocking_spid, NULL) AS input_buffer
WHERE
    DB_NAME(db.database_id) IS NOT NULL
ORDER BY
    db.wait_time_in_seconds DESC,
    db.start_time ASC
OPTION (MAXDOP 1);

-- ========================================
-- SCENARIO 2: UPDATED WAIT ANALYSIS TEST
-- ========================================

-- Test the UPDATED wait analysis query (without SYSTEM_WIDE entries)
DECLARE @TopN INT = 20;
DECLARE @TextTruncateLimit_Wait INT = 4094;

WITH ActiveRequests AS (
    SELECT 
        req.session_id,
        req.request_id,
        req.wait_type,
        req.wait_time,
        req.total_elapsed_time,
        req.database_id,
        COALESCE(
            LEFT(qt.text, @TextTruncateLimit_Wait),
            'Query text not available'
        ) AS query_text,
        req.sql_handle,
        req.start_time
    FROM 
        sys.dm_exec_requests req WITH (NOLOCK)
    OUTER APPLY 
        sys.dm_exec_sql_text(req.sql_handle) qt
    WHERE 
        req.database_id > 4
        AND req.session_id > 50
        AND (
            req.wait_time > 10  -- Currently waiting > 10ms (reduced for testing)
            OR req.total_elapsed_time > 100  -- Long running queries > 100ms
        )
        AND COALESCE(qt.text, '') NOT LIKE '%sp_reset_connection%'
        AND COALESCE(qt.text, '') NOT LIKE '%sys.%'
        AND COALESCE(qt.text, '') NOT LIKE '%INFORMATION_SCHEMA%'
        AND COALESCE(qt.text, '') <> ''
),
CombinedResults AS (
    -- Only current active waiting requests (no SYSTEM_WIDE entries)
    SELECT 
        CONVERT(VARBINARY(255), ar.sql_handle) AS query_id,
        DB_NAME(ar.database_id) AS database_name,
        ar.query_text,
        ar.wait_type AS wait_category,
        CAST(ar.wait_time AS FLOAT) AS total_wait_time_ms,
        CAST(ar.wait_time AS FLOAT) AS avg_wait_time_ms,
        1 AS wait_event_count,
        COALESCE(ar.start_time, SYSDATETIME()) AS last_execution_time,
        SYSDATETIME() AS collection_timestamp
    FROM 
        ActiveRequests ar
    WHERE 
        ar.query_text IS NOT NULL
        AND LEN(LTRIM(RTRIM(ar.query_text))) > 10
        AND DB_NAME(ar.database_id) IS NOT NULL
)
SELECT TOP (@TopN)
    query_id,
    database_name,
    query_text,
    wait_category,
    total_wait_time_ms,
    avg_wait_time_ms,
    wait_event_count,
    last_execution_time,
    collection_timestamp
FROM 
    CombinedResults
WHERE
    database_name IS NOT NULL
ORDER BY 
    total_wait_time_ms DESC
OPTION (MAXDOP 1);

-- ========================================
-- SCENARIO 3: Generate Wait Activity for Testing
-- ========================================

-- Run this to create some wait activity
SELECT 
    p1.BusinessEntityID,
    p1.FirstName,
    p2.LastName,
    COUNT(*) OVER() as total_rows
FROM Person.Person p1
CROSS JOIN Person.Person p2
WHERE p1.BusinessEntityID <= 50
  AND p2.BusinessEntityID <= 50
ORDER BY p1.BusinessEntityID, p2.BusinessEntityID;

-- ========================================
-- Quick Check Queries
-- ========================================

-- 1. Check if there are any currently blocked sessions
SELECT 
    blocking_session_id,
    session_id as blocked_session_id,
    wait_type,
    wait_time,
    command,
    DB_NAME(database_id) as database_name,
    status
FROM sys.dm_exec_requests 
WHERE blocking_session_id > 0
ORDER BY wait_time DESC;

-- 2. Check current active requests with any waits
SELECT 
    session_id,
    status,
    command,
    wait_type,
    wait_time,
    total_elapsed_time,
    DB_NAME(database_id) as database_name
FROM sys.dm_exec_requests 
WHERE session_id > 50
  AND database_id > 4
  AND (wait_time > 0 OR total_elapsed_time > 1000)
ORDER BY wait_time DESC, total_elapsed_time DESC;

-- 3. Check active sessions and their status
SELECT 
    session_id,
    status,
    login_name,
    host_name,
    program_name,
    last_request_start_time,
    last_request_end_time
FROM sys.dm_exec_sessions 
WHERE session_id > 50
  AND status IN ('running', 'sleeping', 'suspended')
ORDER BY last_request_start_time DESC;

-- ========================================
-- CLEANUP
-- ========================================

-- Run this to cleanup the blocking transaction
-- COMMIT; -- or ROLLBACK;
