package migrations

import (
    "context"
    "github.com/PayRpc/Bitcoin-Sprint/internal/database"
    "go.uber.org/zap"
)

// Runner is a minimal migration runner stub
type Runner struct {
    db database.Database
    l  *zap.Logger
}

// NewRunner creates a migration runner
func NewRunner(db database.Database, l *zap.Logger) *Runner {
    return &Runner{db: db, l: l}
}

// Up runs migrations (no-op stub)
func (r *Runner) Up(ctx context.Context) error {
    // no-op in stub
    return nil
}
