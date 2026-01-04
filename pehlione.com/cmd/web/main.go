package main

import (
	"log"
	"os"

	"log/slog"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	apphttp "pehlione.com/app/internal/http"
)

func main() {
	// Load .env file (ignore error if not found - prod uses real env vars)
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Database connection
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	r := apphttp.NewRouter(logger, db)
	_ = r.Run(":8080")
}
