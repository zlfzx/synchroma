## Synchroma
Synchroma is a tool to synchronize database schema from source to target. It will compare the schema and generate a SQL script to sync the schema from source to target.

### Features
- Compare schema from source and target,
- Generate SQL script to sync schema from source to target,
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

### TODO
- [x] Support MySQL
- [x] Compare table schema
- [x] Compare column schema
- [ ] Compare index schema
- [ ] Compare foreign key schema
- [ ] Compare trigger schema
- [ ] Compare view schema
- [ ] Support PostgreSQL and SQLite
- [ ] Add more documentation