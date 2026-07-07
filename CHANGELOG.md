# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-07-07

### Added
- **Dry Run Mode (`--dry-run`)**: Added flag to print the generated SQL to standard output (stdout) without writing to a file.
- **Apply Mode (`--apply`)**: Added flag to execute the generated SQL script directly on the target database on-the-fly.
- **Drop Tables Command (`--drop-tables`)**: Added safety flag to detect and generate `DROP TABLE` statements for tables that exist in the target database but not in the source database.
- **Custom Output File (`--output-file`)**: Added flag to specify a custom file name or path for the generated SQL script.
- **Drop Schema Operations**: Implemented the ability to detect and generate `DROP INDEX` and `DROP FOREIGN KEY` commands for schemas that are no longer in the source database.
- **Unit Testing**: Added initial unit test structure for utility functions (`utils_test.go`).

### Changed
- **Architecture Refactoring**: Introduced the `SchemaProvider` interface. Database logic has been decoupled into `MySQLSchema` to pave the way for supporting multiple databases (like PostgreSQL).
- **Dynamic Versioning**: Changed versioning strategy to be injected dynamically via Go `ldflags` during the build process in the `Makefile`.
- **Column Comparison**: Improved `IsSameColumn` utility function to also compare `OrdinalPosition`, allowing detection of column reordering.
- **Go Version & Dependencies**: Upgraded Go target version to `1.25.3` and updated dependencies including `go-sql-driver/mysql` to `v1.10.0` and `cobra` to `v1.10.2`.

### Fixed
- **SQL Injection Vulnerability**: Fixed an issue where table and column identifiers with special characters or keywords were not safely escaped in the generated DDL. Now strictly applying backticks (`\``).
- **Double DB Close Panic**: Resolved an issue in `handler.go` where `DB.Close()` was being called both via `defer` and manually, preventing potential application panics.
- **Graceful Error Handling**: Replaced aggressive `log.Fatal()` usage within `schema.go` functions with standard `error` return values. The application now exits more gracefully when database reads fail.
- **Redundant Exits**: Cleaned up unnecessary `os.Exit(1)` calls that followed `log.Fatal()` throughout the application.

## [0.1.0] - Initial Release
- Compare schema from source and target (tables, columns, indexes, foreign keys).
- Generate SQL script to sync schema (ADD and MODIFY).
- Basic MySQL support.
- CLI scaffolding with Cobra.
