# Synchroma

**Synchroma** (short for *Synchronize Schema*) is a fast, reliable, and powerful tool designed to **compare and synchronize database schemas** between a source and a target environment. 

Whether you are migrating from development to production or syncing data structures across distributed teams, Synchroma analyzes the differences (Tables, Columns, Indexes, Foreign Keys, Views, Triggers, and Routines) and automatically generates the exact SQL `CREATE`, `ALTER`, and `DROP` scripts needed to make the target database identical to the source.

It fully supports **MySQL** and **PostgreSQL**. It can be used as a standalone CLI tool or imported as a native Go library.

## Installation

### Using Homebrew (macOS / Linux)
```bash
brew tap zlfzx/xyz
brew install synchroma
```

### Using Go
```bash
go install github.com/zlfzx/synchroma@latest
```

## Usage
```bash
synchroma --init
```
configuration file will be created in `~/.synchroma.json`  
or provide configuration parameters directly
```bash
synchroma \
    --database=mysql \
    --source-db-host=source_host \
    --source-db-port=source_port \
    --source-db-user=source_user \
    --source-db-password=source_password \
    --source-db-name=source_db \
    --target-db-host=target_host \
    --target-db-port=target_port \
    --target-db-user=target_user \
    --target-db-password=target_password \
    --target-db-name=target_db

```

You can also use the following flags to control the sync process:
```bash
# Preview the generated SQL script without applying it
synchroma --dry-run

# Directly apply the generated SQL script to the target database
synchroma --apply

# Apply with force (skip destructive operation warnings)
synchroma --apply --force

# Exclude specific tables from sync
synchroma --exclude migrations,sessions,audit_logs

# Only sync specific tables
synchroma --include users,orders,products
```

## Usage as a Go Library
Synchroma is fully decoupled and can be imported directly into your Go backend projects (for example, to build an automated migration service).

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/zlfzx/synchroma/pkg/core"
	"github.com/zlfzx/synchroma/pkg/models"
)

func main() {
	// 1. Define your Source and Target configurations
	sourceDb := models.DataSource{
		Database: "mysql", 
		Host: "localhost", Port: "3306", 
		User: "root", Password: "password", DBName: "db_dev",
	}
	
	targetDb := models.DataSource{
		Database: "mysql", 
		Host: "remote-server.com", Port: "3306", 
		User: "admin", Password: "secure_password", DBName: "db_prod",
	}

	// 2. Setup Synchronization Options
	opts := core.SyncOptions{
		SourceCfg:  sourceDb,
		TargetCfg:  targetDb,
		DropTables: true, // Set to true to drop tables in target that don't exist in source
		
		// Optional: Listen to progress logs directly from the engine
		OnProgress: func(msg string) {
			fmt.Println("[Sync Progress] ->", msg)
		},
	}

	// 3. Generate the SQL
	result, err := core.GenerateSyncSQL(opts)
	if err != nil {
		log.Fatalf("Synchronization failed: %v", err)
	}

	// 4. Use the results!
	fmt.Println("\n--- FINAL SQL SCRIPT ---")
	fmt.Println(result.SQL)
	
	fmt.Printf("\nStatistics: %d tables added, %d modified.\n", 
		result.Stats.TablesAdded, result.Stats.TablesModified)
}
```

### TODO
- [x] Support MySQL
- [x] Support PostgreSQL
- [x] Compare table schema
- [x] Compare column schema
- [x] Compare index schema
- [x] Compare foreign key schema
- [x] Compare trigger schema
- [x] Compare view schema
- [x] Compare routine schema
- [x] Destructive operation warnings
- [x] Table filtering (include/exclude)
- [ ] Add more documentation
