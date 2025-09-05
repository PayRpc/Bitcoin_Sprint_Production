# Bitcoin Sprint - Database Setup

This folder contains everything needed to initialize and manage the Bitcoin Sprint database.

## Files

- `init-db.sql` - SQL script with database schema
- `init-database.ps1` - PowerShell script to run the SQL against PostgreSQL

## Usage

To initialize the database:

```powershell
# Create and initialize a new database
./init-database.ps1 -CreateDb -DbName bitcoin_sprint -DbUser postgres

# Reinitialize an existing database (force recreation)
./init-database.ps1 -CreateDb -Force -DbName bitcoin_sprint -DbUser postgres

# Just run the SQL on an existing database
./init-database.ps1 -DbName bitcoin_sprint -DbUser postgres
```

## CI/CD Integration

In your CI/CD pipeline, you can call this script to set up test databases:

```yaml
- name: Setup Database
  run: ./db/init-database.ps1 -CreateDb -Force -DbName bitcoin_sprint_test -DbUser postgres -DbPassword $DB_PASSWORD
```

## Connection String

For applications, use this connection string format:
```
postgres://user:password@host:port/dbname
```
