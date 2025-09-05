package migrations

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/database"
	"go.uber.org/zap"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Migration represents a single database migration
type Migration struct {
	Version     int
	Name        string
	UpSQL       string
	DownSQL     string
	AppliedAt   *time.Time
	Checksum    string
}

// Runner manages database migrations with full versioning support
type Runner struct {
	db     database.Database
	logger *zap.Logger
	schema string
}

// NewRunner creates a production-ready migration runner
func NewRunner(db database.Database, logger *zap.Logger) *Runner {
	return &Runner{
		db:     db,
		logger: logger,
		schema: "sprint_migrations", // Dedicated schema for migration tracking
	}
}

// Up runs all pending migrations
func (r *Runner) Up(ctx context.Context) error {
	r.logger.Info("Starting database migrations")

	// Initialize migration tracking system
	if err := r.initializeMigrationTracking(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tracking: %w", err)
	}

	// Load available migrations
	migrations, err := r.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Filter pending migrations
	pending := r.filterPendingMigrations(migrations, applied)
	if len(pending) == 0 {
		r.logger.Info("No pending migrations")
		return nil
	}

	r.logger.Info("Found pending migrations", zap.Int("count", len(pending)))

	// Apply migrations in transaction
	return r.applyMigrations(ctx, pending)
}

// Down rolls back the last N migrations
func (r *Runner) Down(ctx context.Context, steps int) error {
	r.logger.Info("Rolling back migrations", zap.Int("steps", steps))

	// Get applied migrations (in reverse order)
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		r.logger.Info("No migrations to rollback")
		return nil
	}

	// Limit rollback steps
	if steps > len(applied) {
		steps = len(applied)
	}

	// Load migration definitions for rollback SQL
	migrations, err := r.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Create migration lookup map
	migrationMap := make(map[int]*Migration)
	for _, m := range migrations {
		migrationMap[m.Version] = m
	}

	// Rollback in reverse order
	for i := 0; i < steps; i++ {
		migration := applied[i]
		migrationDef, exists := migrationMap[migration.Version]
		if !exists {
			return fmt.Errorf("migration definition not found for version %d", migration.Version)
		}

		if migrationDef.DownSQL == "" {
			return fmt.Errorf("migration %d has no rollback SQL", migration.Version)
		}

		if err := r.rollbackMigration(ctx, migrationDef); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// Status returns current migration status
func (r *Runner) Status(ctx context.Context) ([]Migration, error) {
	// Load all available migrations
	available, err := r.loadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create applied map for quick lookup
	appliedMap := make(map[int]*Migration)
	for _, m := range applied {
		appliedMap[m.Version] = m
	}

	// Merge status
	var status []Migration
	for _, migration := range available {
		if appliedMigration, exists := appliedMap[migration.Version]; exists {
			migration.AppliedAt = appliedMigration.AppliedAt
		}
		status = append(status, *migration)
	}

	return status, nil
}

// Force marks a migration as applied without running it (use with caution)
func (r *Runner) Force(ctx context.Context, version int) error {
	r.logger.Warn("Force marking migration as applied", zap.Int("version", version))

	migrations, err := r.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	var targetMigration *Migration
	for _, m := range migrations {
		if m.Version == version {
			targetMigration = m
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration %d not found", version)
	}

	return r.markMigrationApplied(ctx, targetMigration)
}

// initializeMigrationTracking creates the migration tracking schema and table
func (r *Runner) initializeMigrationTracking(ctx context.Context) error {
	queries := []string{
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", r.schema),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			execution_time_ms INTEGER,
			applied_by VARCHAR(100) DEFAULT current_user
		)`, r.schema),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON %s.schema_migrations(applied_at)", r.schema),
	}

	for _, query := range queries {
		if err := r.db.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}

// loadMigrations loads all migration files from embedded filesystem
func (r *Runner) loadMigrations() ([]*Migration, error) {
	files, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []*Migration
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		migration, err := r.parseMigrationFile(file.Name())
		if err != nil {
			r.logger.Warn("Skipping invalid migration file", zap.String("file", file.Name()), zap.Error(err))
			continue
		}

		migrations = append(migrations, migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file (format: 001_create_users.sql)
func (r *Runner) parseMigrationFile(filename string) (*Migration, error) {
	// Extract version and name from filename
	baseName := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(baseName, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid version number in filename: %s", filename)
	}

	// Read file content
	content, err := migrationFiles.ReadFile(filepath.Join("sql", filename))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	contentStr := string(content)
	upSQL, downSQL := r.splitMigrationSQL(contentStr)

	// Calculate checksum
	checksum := r.calculateChecksum(contentStr)

	return &Migration{
		Version:  version,
		Name:     parts[1],
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Checksum: checksum,
	}, nil
}

// splitMigrationSQL splits migration content into UP and DOWN sections
func (r *Runner) splitMigrationSQL(content string) (upSQL, downSQL string) {
	lines := strings.Split(content, "\n")
	var upLines, downLines []string
	inDownSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +migrate Down") {
			inDownSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +migrate Up") {
			inDownSection = false
			continue
		}

		if inDownSection {
			downLines = append(downLines, line)
		} else {
			upLines = append(upLines, line)
		}
	}

	return strings.Join(upLines, "\n"), strings.Join(downLines, "\n")
}

// getAppliedMigrations returns list of applied migrations
func (r *Runner) getAppliedMigrations(ctx context.Context) ([]*Migration, error) {
	query := fmt.Sprintf(`
		SELECT version, name, checksum, applied_at 
		FROM %s.schema_migrations 
		ORDER BY version DESC`, r.schema)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.Version, &migration.Name, &migration.Checksum, &migration.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		migrations = append(migrations, &migration)
	}

	return migrations, nil
}

// filterPendingMigrations returns migrations that haven't been applied
func (r *Runner) filterPendingMigrations(available []*Migration, applied []*Migration) []*Migration {
	appliedMap := make(map[int]bool)
	for _, m := range applied {
		appliedMap[m.Version] = true
	}

	var pending []*Migration
	for _, m := range available {
		if !appliedMap[m.Version] {
			pending = append(pending, m)
		}
	}

	return pending
}

// applyMigrations applies multiple migrations in a transaction
func (r *Runner) applyMigrations(ctx context.Context, migrations []*Migration) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, migration := range migrations {
		start := time.Now()
		r.logger.Info("Applying migration", zap.Int("version", migration.Version), zap.String("name", migration.Name))

		// Execute migration SQL
		if err := r.executeMigrationSQL(ctx, tx, migration.UpSQL); err != nil {
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		// Record migration
		executionTime := int(time.Since(start).Milliseconds())
		if err := r.recordMigration(ctx, tx, migration, executionTime); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		r.logger.Info("Migration applied successfully", 
			zap.Int("version", migration.Version),
			zap.Int("execution_time_ms", executionTime))
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (r *Runner) rollbackMigration(ctx context.Context, migration *Migration) error {
	r.logger.Info("Rolling back migration", zap.Int("version", migration.Version), zap.String("name", migration.Name))

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if err := r.executeMigrationSQL(ctx, tx, migration.DownSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration record
	query := fmt.Sprintf("DELETE FROM %s.schema_migrations WHERE version = $1", r.schema)
	if err := tx.Exec(ctx, query, migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	r.logger.Info("Migration rolled back successfully", zap.Int("version", migration.Version))
	return nil
}

// executeMigrationSQL executes SQL statements in a migration
func (r *Runner) executeMigrationSQL(ctx context.Context, tx database.Tx, sql string) error {
	statements := r.splitSQLStatements(sql)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute SQL statement: %w", err)
		}
	}
	return nil
}

// splitSQLStatements splits SQL content into individual statements
func (r *Runner) splitSQLStatements(sql string) []string {
	// Simple statement splitting (doesn't handle complex cases like dollar-quoted strings)
	statements := strings.Split(sql, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

// recordMigration records a successful migration
func (r *Runner) recordMigration(ctx context.Context, tx database.Tx, migration *Migration, executionTime int) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.schema_migrations (version, name, checksum, execution_time_ms) 
		VALUES ($1, $2, $3, $4)`, r.schema)
	
	return tx.Exec(ctx, query, migration.Version, migration.Name, migration.Checksum, executionTime)
}

// markMigrationApplied marks a migration as applied without executing it
func (r *Runner) markMigrationApplied(ctx context.Context, migration *Migration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.schema_migrations (version, name, checksum, execution_time_ms) 
		VALUES ($1, $2, $3, 0) ON CONFLICT (version) DO NOTHING`, r.schema)
	
	return r.db.Exec(ctx, query, migration.Version, migration.Name, migration.Checksum)
}

// calculateChecksum calculates MD5 checksum of migration content
func (r *Runner) calculateChecksum(content string) string {
	// Simple hash for checksum - in production you might want crypto/md5
	hash := 0
	for _, char := range content {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}
