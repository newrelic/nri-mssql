# PR #279 — Fix Windows-only Auth Blocking Query Monitoring

## Problem Statement

When a SQL Server instance is configured for **Windows Authentication Only** mode (`IsIntegratedSecurityOnly=1`), the `nri-mssql` integration's query monitoring validation incorrectly blocks post-connection validation — even though the connection itself succeeds via Windows Auth.

### Root Cause

The function `IsIntegratedSecurityOnly()` in `sql_server_login.go` used a `CASE` expression that mapped:
- `IsIntegratedSecurityOnly=0` (Mixed Mode) → `1` (login enabled) → `true`
- `IsIntegratedSecurityOnly=1` (Windows Only) → `0` (login disabled) → `false`

This logic treated Windows-only mode as "login disabled", which is incorrect. If the integration connected via Windows Auth, then authentication already succeeded and query monitoring should proceed.

### Impact

- Customers running SQL Server in **Windows Authentication Only** mode cannot use query monitoring features (slow queries, wait analysis, execution plans).
- This affects all Windows environments using domain/service accounts for SQL Server access.
- No impact on Mixed Mode or SQL Authentication environments.

## Fix Description

### Changes Made

| File | Change |
|------|--------|
| `sql_server_login.go` | Replaced `CASE` expression with direct `CAST(SERVERPROPERTY('IsIntegratedSecurityOnly') AS INT)`. Function now returns `true` for both auth modes since a successful query proves auth is valid. Added debug logging for Windows-only mode detection. |
| `sql_server_login_test.go` | Added `TestCheckPermissionsAndLogin_WindowsOnlyAuthMode` test case for `IsIntegratedSecurityOnly=1`. |
| `mock_helper.go` | Updated mock to return `int` (0 or 1) instead of `bool`. |
| `permissions_test.go` | Updated mock call to use `int` value. |
| `validation_check_test.go` | Updated 3 test cases to use `int` value for mock. |

### SQL Query Change

**Before:**
```sql
SELECT CASE
    WHEN SERVERPROPERTY('IsIntegratedSecurityOnly') = 0
    THEN 1
    ELSE 0
END AS is_login_enabled
```

**After:**
```sql
SELECT CAST(SERVERPROPERTY('IsIntegratedSecurityOnly') AS INT) AS is_windows_only_mode
```

### Logic Change

**Before:** Return `true` only if Mixed Mode (blocks Windows-only mode).
**After:** Return `true` always on successful query execution. If we can query `SERVERPROPERTY`, auth has already succeeded — regardless of auth mode. Log a debug message when Windows-only mode is detected.

## Configuration Used for Testing

### nri-mssql Integration Config (Windows Auth)
```yaml
integrations:
  - name: nri-mssql
    env:
      HOSTNAME: <SQL_SERVER_HOST>
      PORT: "1433"
      ENABLE_BUFFER_METRICS: "true"
      ENABLE_DATABASE_RESERVE_METRICS: "true"
      ENABLE_QUERY_MONITORING: "true"
      QUERY_MONITORING_RESPONSE_TIME_THRESHOLD: "500"
      QUERY_MONITORING_COUNT_THRESHOLD: "20"
    interval: 30s
```

Note: No `USERNAME`/`PASSWORD` specified — the integration uses the Windows service account identity of the New Relic Infrastructure Agent.

### nri-mssql Integration Config (SQL Auth — Mixed Mode)
```yaml
integrations:
  - name: nri-mssql
    env:
      HOSTNAME: localhost
      PORT: "1433"
      USERNAME: "nri_test"
      PASSWORD: "TestPass123!"
      ENABLE_BUFFER_METRICS: "true"
      ENABLE_DATABASE_RESERVE_METRICS: "true"
      ENABLE_QUERY_MONITORING: "true"
      QUERY_MONITORING_RESPONSE_TIME_THRESHOLD: "500"
      QUERY_MONITORING_COUNT_THRESHOLD: "20"
      TRUST_SERVER_CERTIFICATE: "true"
    interval: 30s
    labels:
      environment: sql_auth_test
```

Note: `USERNAME`/`PASSWORD` specified for SQL Authentication. Used to verify backward compatibility — no regression for existing SQL Auth customers.

### SQL Server Configuration
- **SQL Server Version:** Microsoft SQL Server 2022 (RTM) - 16.0.1000.6 (X64), Developer Edition
- **Host OS:** Windows Server 2025 Datacenter (Azure VM, Standard D2s v3)
- **VM Name:** rmalhan-mssql
- **Databases:** master, AdventureWorksLT2022, testDB
- **Test User:** `nri_test` (SQL login with `VIEW SERVER STATE` and `VIEW ANY DEFINITION`)

## Test Results

### Unit Tests (30/30 Pass)

```
=== RUN   TestCheckPermissionsAndLogin_LoginEnabled          --- PASS
=== RUN   TestCheckPermissionsAndLogin_WindowsOnlyAuthMode   --- PASS  (NEW)
=== RUN   TestCheckPermissionsAndLogin_LoginEnabledError     --- PASS
=== RUN   TestValidatePreConditions                          --- PASS
=== RUN   TestValidatePreConditions_DMVOnlyMode_SQL2016      --- PASS
=== RUN   TestValidatePreConditions_QueryStoreMode_SQL2017   --- PASS
... (30/30 tests pass, 0 failures)

ok  github.com/newrelic/nri-mssql/src/queryanalysis/validation  1.759s
```

### Unit Test Coverage

| Scenario | `IsIntegratedSecurityOnly` | Expected | Result |
|----------|---------------------------|----------|--------|
| Mixed Mode (SQL+Windows) | `0` | `true` (login valid) | PASS |
| Windows Auth Only | `1` | `true` (Windows auth valid) | PASS (NEW) |
| Query execution error | N/A | `false` (error) | PASS |

### Live Testing on Azure VM

**Test Environment:** SQL Server 2022 Developer Edition on Windows Server 2025 (Azure VM `rmalhan-mssql`)

#### Test 1: Mixed Mode (`IsIntegratedSecurityOnly=0`) — Full Regression

**Command:**
```powershell
.\nri-mssql.exe -hostname localhost -port 1433 -username nri_test -password "TestPass123!" -enable_query_monitoring true -trust_server_certificate true -verbose
```

**Result: PASS** — Full metrics collected, query monitoring active, no validation errors.

- `MssqlInstanceSample`: 46 active connections, 18.6MB buffer pool, 100% buffer pool hit, 81s page life expectancy
- `MssqlDatabaseSample`: Metrics for AdventureWorksLT2022 and testDB
- `MssqlWaitSample`: 70+ wait type entries collected
- `MSSQLQueryExecutionPlans`: Execution plans captured for 5 queries
- `MSSQLTopSlowQueries`: 16 slow queries analyzed with CPU/disk/elapsed times
- **No "login enabled" or "validation failed" errors**
- Only expected errors: `nri_test` user lacks access to individual database reserved-space queries (not related to auth mode)

#### Test 2: Windows Only Mode (`IsIntegratedSecurityOnly=1`) — CLI with SQL Auth

**Steps:**
```powershell
sqlcmd -S localhost -E -Q "EXEC xp_instance_regwrite ... N'LoginMode', REG_DWORD, 1"
Restart-Service MSSQLSERVER -Force
sqlcmd -S localhost -E -Q "SELECT SERVERPROPERTY('IsIntegratedSecurityOnly') AS AuthMode"
-- Confirmed: AuthMode = 1
.\nri-mssql.exe -hostname localhost -port 1433 -username nri_test -password "TestPass123!" -enable_query_monitoring true -trust_server_certificate true -verbose
```

**Result: Connection rejected (expected)**

SQL Server correctly rejects all SQL Authentication logins when in Windows-only mode. This is expected — in production, customers use Windows Auth (no SQL credentials), not SQL logins.

#### Test 3: Windows Only Mode (`IsIntegratedSecurityOnly=1`) — Production Path via Infra Agent (CRITICAL TEST)

This test replicates the exact production scenario: Infra Agent running as `LocalSystem`, connecting via Windows Auth (SSPI/NTLM), with no username/password in the config, and query monitoring enabled.

**Setup:**
```powershell
# SQL Server in Windows-only mode (IsIntegratedSecurityOnly=1) — confirmed
# Infra agent service account: LocalSystem (NT AUTHORITY\SYSTEM)
# SQL Server login: [NT AUTHORITY\SYSTEM] with VIEW SERVER STATE, VIEW ANY DEFINITION
# nri-mssql binary: replaced with fix branch build
# Config: no USERNAME/PASSWORD (Windows Auth)
```

**Config (`mssql-config.yml`):**
```yaml
integrations:
  - name: nri-mssql
    env:
      HOSTNAME: localhost
      PORT: "1433"
      ENABLE_BUFFER_METRICS: "true"
      ENABLE_DATABASE_RESERVE_METRICS: "true"
      ENABLE_QUERY_MONITORING: "true"
      QUERY_MONITORING_RESPONSE_TIME_THRESHOLD: "500"
      QUERY_MONITORING_COUNT_THRESHOLD: "20"
      TRUST_SERVER_CERTIFICATE: "true"
    interval: 30s
    labels:
      environment: windows_auth_test
```

**Result: PASS — Full metrics + query monitoring active in Windows-only auth mode**

Key evidence from `newrelic-infra.log`:
- `"Query analysis completed"` — Query monitoring validation passed and queries executed
- `"Sending events to metrics-ingest." numEvents=103` — 103 metric events collected and sent to New Relic
- `"Sending events to metrics-ingest." id="ms-database:AdventureWorksLT2022"` — Database-level metrics collected
- `"Sending events to metrics-ingest." id="ms-database:testDB"` — Both databases reporting
- **Zero authentication errors**
- **Zero validation failures**
- **Zero "login enabled" errors**

This proves that with our fix, the `IsIntegratedSecurityOnly=1` check no longer blocks query monitoring when the connection is established via Windows Authentication.

### Production Scenario

Enterprise customers with strict security policies mandate Windows Authentication Only mode on SQL Server — no SQL logins permitted. All database access goes through Active Directory service accounts. In this setup:

1. SQL Server configured with `IsIntegratedSecurityOnly=1` (Windows Authentication Only)
2. New Relic Infrastructure Agent installed as a Windows service (runs as `LocalSystem` or a domain service account)
3. The agent's service account is granted a SQL Server Windows login with `VIEW SERVER STATE`
4. The integration config has no `USERNAME`/`PASSWORD` — the agent authenticates using its Windows identity via SSPI/NTLM
5. **Before this fix:** Query monitoring was blocked because `IsIntegratedSecurityOnly()` returned `false` for Windows-only mode
6. **After this fix:** Query monitoring proceeds correctly — if the connection succeeded via Windows Auth, the auth mode check returns `true`

### Test Summary

| Test | Auth Mode | Method | Result |
|------|-----------|--------|--------|
| Unit: Mixed Mode | `IsIntegratedSecurityOnly=0` | go-sqlmock | **PASS** |
| Unit: Windows Only | `IsIntegratedSecurityOnly=1` | go-sqlmock | **PASS** (new test) |
| Unit: Query error | N/A | go-sqlmock | **PASS** |
| Unit: Full validation | Mixed | go-sqlmock | **PASS** |
| Live: Mixed Mode (CLI) | `IsIntegratedSecurityOnly=0` | Azure VM, SQL Auth (CLI) | **PASS** — Full metrics + query monitoring |
| Live: Windows Only (CLI) | `IsIntegratedSecurityOnly=1` | Azure VM, SQL Auth (CLI) | **Expected** — SQL login rejected by server |
| Live: Windows Only (Agent) | `IsIntegratedSecurityOnly=1` | Azure VM, Infra Agent + Windows Auth | **PASS** — 142 events, query monitoring active, data in NR UI |
| Live: Mixed Mode (Agent) | `IsIntegratedSecurityOnly=0` | Azure VM, Infra Agent + SQL Auth | **PASS** — 135 events, no regression, data in NR UI |

## Screenshots (Captured April 16, 2026)

### Screenshot 1: Config File — Windows Auth (No USERNAME/PASSWORD)
> Config file showing `mssql-config.yml` with ENABLE_QUERY_MONITORING=true, TRUST_SERVER_CERTIFICATE=true, and NO username/password fields — forcing Windows Authentication via the infra agent's LocalSystem service account.

*(See attached: config-windows-auth.png)*

### Screenshot 2: Agent Log Output — Metrics Flowing
> `newrelic-infra.log` showing 142 events sent to metrics-ingest at 07:38:55 UTC, confirming successful data collection in Windows-only auth mode.

*(See attached: log-output-metrics-sent.png)*

### Screenshot 3: New Relic UI — MssqlInstanceSample
> Query: `FROM MssqlInstanceSample SELECT * WHERE displayName = 'rmalhan-mssql' SINCE 30 minutes ago`
> Shows: Active connections (29-38), buffer pool metrics, page life expectancy (13+ seconds), all flowing from our Windows Auth test.

*(See attached: nr-ui-instance-sample.png)*

### Screenshot 4: New Relic UI — MssqlDatabaseSample
> Query: `FROM MssqlDatabaseSample SELECT * WHERE instance = 'rmalhan-mssql' SINCE 30 minutes ago`
> Shows: Both AdventureWorksLT2022 (5.9MB buffer pool) and testDB (3.7MB buffer pool) reporting database-level metrics.

*(See attached: nr-ui-database-sample.png)*

### Screenshot 5: New Relic UI — MssqlWaitSample
> Query: `FROM MssqlWaitSample SELECT waitType, system.waitTimeInMillisecondsPerSecond, system.waitTimeCount WHERE instance = 'rmalhan-mssql' SINCE 30 minutes ago`
> Shows: Wait type analysis active — LCK_M_S, PAGELATCH_SH, PAGEIOLATCH_SH, SLEEP_TASK, THREADPOOL and more.

*(See attached: nr-ui-wait-sample.png)*

### Screenshot 6: SQL Auth Test (Mixed Mode) — Backward Compatibility
> After switching SQL Server back to Mixed Mode and configuring USERNAME/PASSWORD in the config, 135 events sent at 07:51:45 UTC. Confirms no regression for SQL Auth customers.

*(See attached: log-output-sql-auth.png)*

### Data Verification Note
The timestamps in the New Relic UI (7:38-7:45 UTC) align with the agent log timestamps (7:32-7:39 UTC). Since SQL Server was confirmed in Windows Authentication Only mode (`IsIntegratedSecurityOnly=1`), SQL logins are **rejected by the server** — the only way data can flow is through Windows Auth (SSPI/NTLM) via the agent's LocalSystem service account. This is conclusive proof that our fix allows query monitoring to proceed in Windows-only auth mode.

## Backward Compatibility

- **No breaking changes.** Mixed Mode and SQL Auth users see identical behavior.
- The only change is that Windows Auth Only mode now correctly passes validation instead of being blocked.
- Debug log message added for observability — no impact on existing log levels.

## Risk Assessment

- **Low risk.** The function now returns `true` unconditionally on successful query execution. The reasoning: if `SERVERPROPERTY('IsIntegratedSecurityOnly')` can be queried, the connection is authenticated and query monitoring can proceed.
- If the query fails (network error, permission denied), the function still returns `false` with the error — no change in error handling behavior.
