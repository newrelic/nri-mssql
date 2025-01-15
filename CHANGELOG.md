# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

Unreleased section should follow [Release Toolkit](https://github.com/newrelic/release-toolkit#render-markdown-and-update-markdown)

## Unreleased

## v2.16.0 - 2024-12-19

### 🚀 Enhancements
- Updated golang.org/x/net to v0.33.0

## v2.15.0 - 2024-12-17

### 🚀 Enhancements
- Updated goreleaser to v2.4.4
- Added FIPS compliant packages and archives

## v2.14.1 - 2024-12-17

### dependency
- Updated golang.org/x/crypto to v0.31.0
- Updated golang.org/x/text to v0.21.0

### ⛓️ Dependencies
- Updated golang patch version to v1.23.4
- Updated github.com/microsoft/go-mssqldb to v1.8.0 - [Changelog 🔗](https://github.com/microsoft/go-mssqldb/releases/tag/v1.8.0)

## v2.14.0 - 2024-12-03

### 🚀 Enhancements
- Add `enable_disk_metrics_in_bytes` argument (default true), allows disabling disk-space-metrics

### ⛓️ Dependencies
- Updated golang patch version to v1.23.3

## v2.13.0 - 2024-10-08

### dependency
- Upgrade go to 1.23.2

### 🚀 Enhancements
- Upgrade integrations SDK so the interval is variable and allows intervals up to 5 minutes

## v2.12.8 - 2024-09-17

### 🐞 Bug fixes
- Handle DECIMAL values in custom queries

## v2.12.7 - 2024-09-10

### ⛓️ Dependencies
- Updated golang version to v1.23.1
- (deps) move from github.com/denisenkom/go-mssqldb to github.com/microsoft/go-mssqldb

## v2.12.6 - 2024-07-09

### ⛓️ Dependencies
- Updated golang version to v1.22.5

## v2.12.5 - 2024-06-18

### 🐞 Bug fixes
- Avoid int overflow for big numeric values

## v2.12.3 - 2024-05-14

### ⛓️ Dependencies
- Updated golang version to v1.22.3

## v2.12.2 - 2024-04-30

### ⛓️ Dependencies
- Updated github.com/jmoiron/sqlx to v1.4.0 - [Changelog 🔗](https://github.com/jmoiron/sqlx/releases/tag/v1.4.0)

## v2.12.1 - 2024-04-16

### ⛓️ Dependencies
- Updated golang version to v1.22.2

## v2.12.0 - 2024-02-22

### 🚀 Enhancements
- Improve debug logs

### ⛓️ Dependencies
- Updated github.com/newrelic/infra-integrations-sdk to v3.8.2+incompatible

## v2.11.0 - 2024-02-07

### 🚀 Enhancements
- Improve performance of query listing `instance_active_connections` by only counting on `sys.sysprocesses`.

## v2.10.3 - 2024-01-30

### ⛓️ Dependencies
- Updated golang.org/x/crypto

## v2.10.2 - 2023-10-31

### ⛓️ Dependencies
- Updated golang version to 1.21

## v2.10.1 - 2023-08-08

### ⛓️ Dependencies
- Updated golang to v1.20.7

## v2.10.0 - 2023-07-25

### 🚀 Enhancements
- bumped golang version pinning 1.20.6

## 2.9.0 (2023-06-06)
### Changed
- Update Go version to 1.20

## 2.8.7  (2022-12-31)
### Changed
- Modified bufferPoolHitPercent to avoid issue during the computation of the ratio.
- Updated dependencies and go version

## 2.8.6  (2022-10-03)
### Changed
- Optimized Buffer Pool queries for additional performance. Issue [#82](https://github.com/newrelic/nri-mssql/issues/82)

## 2.8.5  (2022-09-15)
### Changed
- Fixed issue parsing custom-queries results

## 2.8.4  (2022-08-17)
### Changed
- Avoid potential deadlocks in disk space query

## 2.8.3  (2022-08-17)
### Changed
- Improve error handling and debug logs for custom queries

## 2.8.2  (2022-06-27)
### Changed
- Bump dependencies
### Added
Added support for more distributions:
- RHEL(EL) 9
- Ubuntu 22.04

## 2.8.1 (2021-10-20)
### Added
Added support for more distributions:
- Debian 11
- Ubuntu 20.10
- Ubuntu 21.04
- SUSE 12.15
- SUSE 15.1
- SUSE 15.2
- SUSE 15.3
- Oracle Linux 7
- Oracle Linux 8

## 2.8.0 (2021-08-27)
### Changed

Moved default config.sample to [V4](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-newer-configuration-format/), added a dependency for infra-agent version 1.20.0

Please notice that old [V3](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-standard-configuration-format/) configuration format is deprecated, but still supported.

## 2.7.1 (2021-08-01)
### Fixed
- Fixing issue related to denisenkom/go-mssqldb#639

## 2.6.2 (2021-07-09)
### Fixed
- Ignore model_msdb and model_replicatedmaster system databases (#72)

## 2.6.1 (2021-06-08)
### Changed
- Support for ARM

## 2.6.0 (2021-06-05)
### Changed
- Update Go to v1.16.
- Migrate to Go Modules
- Update Infrastracture SDK to v3.6.7.
- Update other dependecies.
## 2.5.4 (2021-04-06)
### Added
- `ExtraConnectionURLArgs` argument allowing to specify custom connection strings

## 2.5.3 (2021-03-23)
### Changes
- Adds arm packages and binaries

## 2.5.2 (2020-11-16)
### Fixed
- Add connection timeout to connection params to mitigate a zombie request bug in the driver

## 2.5.1 (2020-07-29)
### Fixed
- MSSQL instances were being reported with only the host name instead of the full instance name

## 2.5.0 (2020-07-13)
### Changed
- Updated the MSSQL driver
- Username is no longer required to open up support for Windows auth

## 2.4.2 (2020-07-13)
### Changed
- Rename bufferPoolHit to bufferPoolHitPercent

## 2.4.1 (2020-03-12)
### Changed
- Skip system databases that we don't get permissions for by default

## 2.4.0 (2020-03-05)
### Added
- `EnableDatabaseReserveMetrics` argument

## 2.3.1 (2020-02-12)
### Fixed
- Missing vendored dependency

## 2.3.0 (2020-02-11)
### Added
- Support for custom metrics query file with `custom_metrics_config`

## 2.2.1 (2020-01-13)
### Fixed
- Make sample query a valid MSSQL query

## 2.2.0 (2020-01-13)
### Added
- Support for custom metrics with `custom_metrics_query`

## 2.1.0 (2019-11-18)
### Changed
- Renamed the integration executable from nr-mssql to nri-mssql in order to be consistent with the package naming. **Important Note:** if you have any security module rules (eg. SELinux), alerts or automation that depends on the name of this binary, these will have to be updated.

## 2.0.7 - 2019-11-11
### Changed
- Add `enable_buffer_metrics` (default true) option, which allows disabling resource-intensive buffer metrics

## 2.0.6 - 2019-09-26
### Fixed
- Add instance name fallbacks with COALESCE

## 2.0.4 - 2019-09-16
### Fixed
- Add NOLOCK hints to avoid deadlocking

## 2.0.3 - 2019-07-30
### Changed
- Windows build scripts for packaging

## 2.0.2 - 2019-07-17
### Changed
- Fixed bug causing host to be collected as a database

## 2.0.0 - 2019-05-06
### Changed
- Updated SDK
- Made entity keys more unique

## 1.1.2 - 2019-02-04
### Changed
- Updated Definition file protocol version to 2

## 1.0.1 - 2018-11-29
### Changes
- Fixed MSI install location

## 1.0.0 - 2018-11-29
### Changes
- Bumped version for GA release

## 0.1.5 - 2018-11-15
### Added
- Instance as an attribute to WaitGroup and Instance samples
- Host as an attribute to all samples

## 0.1.4 - 2018-11-14
### Changed
- Made sub query for Instance errors more generic

## 0.1.3 - 2018-11-14
### Fixed
- Issue where if no rows were returned for an instance query then a panic would occur

## 0.1.2 - 2018-11-08
### Changed
- If both port and instance are not specified will default to port 1433

## 0.1.1 - 2018-10-18
### Removed
- Comment from definition file

## 0.1.0 - 2018-09-20
### Added
- Initial version: Includes Metrics and Inventory data
