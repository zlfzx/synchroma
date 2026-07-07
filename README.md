## Synchroma
Synchroma is a tool to synchronize database schema from source to target. It will compare the schema and generate a SQL script to sync the schema from source to target.

### Features
- Compare schema from source and target,
- Generate SQL script to sync schema from source to target (including Tables, Columns, Indexes, Foreign Keys, Views, Triggers, and Routines),
- Support MySQL, ~~PostgreSQL, and SQLite~~

### Installation
```bash
brew tap zlfzx/xyz
brew install synchroma
```

### Usage

initialize configuration file before sync schema
```bash
synchroma --init
```
configuration file will be created in `~/.synchroma`  
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
```

### TODO
- [x] Support MySQL
- [x] Compare table schema
- [x] Compare column schema
- [x] Compare index schema
- [x] Compare foreign key schema
- [x] Compare trigger schema
- [x] Compare view schema
- [x] Compare routine schema
- [ ] Support PostgreSQL
- [ ] Add more documentation
