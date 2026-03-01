package db

import (
	"fmt"
	"log"
	"time"

	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/creative-computing-society/codeboard/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		config.C.DBHost, config.C.DBUser, config.C.DBPass, config.C.DBName, config.C.DBPort,
	)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Warn),
		PrepareStmt: true, // cache prepared statements — eliminates repeated parse/plan overhead
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Tune the underlying sql.DB pool.
	// RefreshUserData spawns one goroutine per user, so cap open conns to avoid
	// exhausting Postgres's max_connections.
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	log.Println("Database connected successfully")
}

func Migrate() {
	err := DB.AutoMigrate(
		&models.CUser{},
		&models.Leetcode{},
		&models.Question{},
		&models.LeaderboardEntry{},
		&models.Leaderboard{},
	)
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Database migrated successfully")
}
