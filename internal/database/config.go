package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"groops/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() {
	// Database connection parameters
	host := getEnvOrDefault("DB_HOST", "localhost")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbname := getEnvOrDefault("DB_NAME", "groops")
	port := getEnvOrDefault("DB_PORT", "5432")

	// Connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second, // Log queries slower than 1 second
				LogLevel:                  logger.Info, // Log all SQL queries
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Enable color
			},
		),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table names
		},
		PrepareStmt:                              true,  // Enable prepared statement cache
		SkipDefaultTransaction:                   false, // Keep default transaction for safety
		DisableForeignKeyConstraintWhenMigrating: false, // Enable foreign key constraints
	}

	// Open connection
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
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

	// Auto Migrate the schema
	err = db.AutoMigrate(
		&models.Account{},
		&models.Group{},
		&models.GroupMember{},
		&models.ActivityLog{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	DB = db
	log.Println("Database connection established")
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// WithTx executes function within a transaction
//not being used currently
// func WithTx(fn func(tx *gorm.DB) error) error {
// 	return DB.Transaction(fn)
// }
