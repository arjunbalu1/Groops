package database

import (
	"fmt"
	"groops/internal/models"
	"groops/internal/utils"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() error {
	var dsn string

	// Check if we're in production mode
	if os.Getenv("GIN_MODE") == "release" {
		// In production, use the Railway DATABASE_URL
		dsn = getEnvRequired("DATABASE_URL")
	} else {
		// In development, use individual connection parameters
		host := getEnvRequired("DB_HOST")
		user := getEnvRequired("DB_USER")
		password := getEnvRequired("DB_PASSWORD")
		dbname := getEnvRequired("DB_NAME")
		port := getEnvRequired("DB_PORT")
		sslMode := os.Getenv("DB_SSL_MODE")
		if sslMode == "" {
			sslMode = "disable" // Default to disable for local development
		}

		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC connect_timeout=10",
			host, user, password, dbname, port, sslMode)
	}

	// Create base logger
	baseLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags|log.Lshortfile),
		logger.Config{
			SlowThreshold:             time.Second, // Log queries slower than 1 second
			LogLevel:                  logger.Info, // Keep logging all SQL queries
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)

	// Create custom logger that filters specific queries
	customLogger := utils.NewCustomGormLogger(
		baseLogger,
		"SELECT * FROM \"group\" WHERE date_time >", // Filter reminder worker query
	)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: customLogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names
		},
		PrepareStmt:                              true,  // Enable prepared statement cache
		SkipDefaultTransaction:                   false, // Keep default transaction for safety
		DisableForeignKeyConstraintWhenMigrating: false, // Enable foreign key constraints
	}

	// Open connection with retry logic
	var err error
	maxRetries := 5
	retryDelay := time.Second * 5

	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			break
		}
		log.Printf("Database connection attempt %d failed: %v", i+1, err)
		if i < maxRetries-1 {
			log.Printf("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)           // Maximum number of idle connections
	sqlDB.SetMaxOpenConns(100)          // Maximum number of open connections
	sqlDB.SetConnMaxLifetime(time.Hour) // Maximum lifetime of a connection

	if err := DB.AutoMigrate(
		&models.Account{},
		&models.Group{},
		&models.GroupMember{},
		&models.ActivityLog{},
		&models.Notification{},
		&models.Session{},
		&models.LoginLog{},
		&models.ReminderSent{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connection established and migrations completed")
	return nil
}

// getEnvRequired returns environment variable value or panics if not set
func getEnvRequired(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("Required environment variable %s is not set", key)
	return "" // This line will never execute due to the log.Fatalf above
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
