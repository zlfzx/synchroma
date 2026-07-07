# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-07-07

### Added
- **Go Library Support**: The core synchronization engine has been fully decoupled from the CLI. Synchroma can now be imported and used programmatically as a native Go library in other projects via the `synchroma/pkg/core` and `synchroma/pkg/config` packages.
- **Advanced Schema Objects**: Added sync support for `VIEWS`, `TRIGGERS`, and `ROUTINES` (Stored Procedures & Functions). Synchroma now detects changes and generates `DROP` and `CREATE` scripts for these objects.
- **Table Properties Sync**: Synchroma now detects and synchronizes changes to table properties such as `ENGINE`, `COLLATE`, and `COMMENT`.
- **Multi-Profile Configuration**: Introduced a new `~/.synchroma.json` configuration file supporting multiple connection profiles (e.g., `staging`, `production`). Added the `--profile` flag to switch between them.
- **Execution Summary**: Added a clean summary block at the end of the sync process, detailing exactly how many tables, columns, indexes, and complex objects were added, modified, or dropped, along with execution time.

### Changed
- **Directory Restructuring**: Relocated all core logic from the restricted `internal/` directory to the public `pkg/` directory (`pkg/core`, `pkg/config`, `pkg/models`, `pkg/schema`, `pkg/utils`) to comply with Go library standards.
- **CLI Decoupling**: The Cobra CLI layer (`cmd/root.go`) has been cleanly separated to act as a client consuming the new library engine, removing all CLI dependencies from the core database synchronization logic.
- **Callback-Based Logging**: Replaced aggressive `fmt.Println` outputs inside the core engine with a flexible `OnProgress` callback mechanism (`SyncOptions.OnProgress`), ensuring clean execution without polluting `stdout` for library consumers.
- **Parallel Processing**: Dramatically improved execution speed by processing table comparisons concurrently using Go routines. Includes a bounded semaphore (max 10 concurrent routines) to protect database connection pools from being overloaded.
- **Thread-Safe Architecture**: Restructured the internal statistics and logging mechanisms using `sync.Mutex` to prevent race conditions and overlapping text output during parallel execution.
- **Config Strategy**: Completely removed the dependency on `godotenv` (`.env` files) in favor of the standard library `encoding/json`.

### Fixed
- **Memory Safety**: Resolved `go vet` compiler errors regarding `sync.Mutex` copy locks by properly passing the `SyncStats` struct via pointers across the application.

### Removed
- Removed `.env` initialization logic and the `godotenv` package.


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
