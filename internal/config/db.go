package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maharanidw/translator-pipeline/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_URL") // support DB_URL naming
	}

	// Jika menggunakan URL dari Supabase Pooler (Pgbouncer),
	// kita WAJIB menghilangkan parameter prepare statement
	// agar tidak terjadi ERROR "prepared statement already exists" (SQLSTATE 42P05)
	var gormDB *gorm.DB
	var err error

	// GORM Config opsional: log hanya jika ada error (agar terminal lebih rapi)
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	}

	fmt.Println("\n=========== Setup Database ===========")
	fmt.Println("Connecting to database...")

	if dsn != "" {
		// Mengakali kebiasaan Session Pooler Supabase (Port 6543)
		// Kita mematikan PreferSimpleProtocol=true agar tidak gagal auto-migrate
		// akibat Transaction Pooler.
		db, err := gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // WAJIB TRUE untuk connection pooler spt PgBouncer / Supabase
		}), gormConfig)

		if err != nil {
			log.Fatalf("failed connect to database PostgreSQL via DSN: %v", err)
		}
		gormDB = db

	} else {
		// Fallback ke metode lama jika DATABASE_URL tidak ada (Lokal)
		fallbackDsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
			os.Getenv("DB_HOST"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_PORT"),
		)
		db, err := gorm.Open(postgres.Open(fallbackDsn), gormConfig)
		if err != nil {
			log.Fatal("failed connect to database PostgreSQL fallback:", err)
		}
		gormDB = db
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatal("failed to get object sql.DB:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// --- AUTO MIGRATE ---
	log.Println("Auto migrating Database...")
	err = gormDB.AutoMigrate(
		&models.Novel{},
		&models.Chapter{},
		&models.Glossary{},
		&models.DailyUsage{},
	)
	if err != nil {
		log.Fatal("failed auto-migrate database:", err)
	}

	DB = gormDB
	log.Println("Migrate SUCCESS || Database Connected")
}
