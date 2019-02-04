# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

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