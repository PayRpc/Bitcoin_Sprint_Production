package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PayRpc/Bitcoin-Sprint/internal/database"
	"github.com/PayRpc/Bitcoin-Sprint/internal/migrations"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Check for database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/bitcoin_sprint_test?sslmode=disable"
		fmt.Printf("Using default database URL: %s\n", dbURL)
	}

	// Create database connection
	dbConfig := database.Config{
		Type: "postgres",
		URL:  dbURL,
		MaxConns: 10,
		MinConns: 2,
	}

	db, err := database.New(dbConfig, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test migrations
	migrationRunner := migrations.NewRunner(db, logger)
	
	fmt.Println("📊 Migration Status:")
	status, err := migrationRunner.Status(context.Background())
	if err != nil {
		logger.Error("Failed to get migration status", zap.Error(err))
		return
	}

	for _, migration := range status {
		statusStr := "❌ PENDING"
		if migration.AppliedAt != nil {
			statusStr = fmt.Sprintf("✅ APPLIED (%s)", migration.AppliedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  %03d_%s: %s\n", migration.Version, migration.Name, statusStr)
	}

	// Run migrations
	fmt.Println("\n🔄 Running migrations...")
	if err := migrationRunner.Up(context.Background()); err != nil {
		logger.Error("Migration failed", zap.Error(err))
		return
	}

	fmt.Println("\n📊 Final Migration Status:")
	status, err = migrationRunner.Status(context.Background())
	if err != nil {
		logger.Error("Failed to get final migration status", zap.Error(err))
		return
	}

	for _, migration := range status {
		statusStr := "❌ PENDING"
		if migration.AppliedAt != nil {
			statusStr = fmt.Sprintf("✅ APPLIED (%s)", migration.AppliedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  %03d_%s: %s\n", migration.Version, migration.Name, statusStr)
	}

	fmt.Println("\n🎉 Migration test completed successfully!")
}
