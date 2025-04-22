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
func InitDB() error {
	// Database connection parameters from environment variables
	host := getEnvRequired("DB_HOST")
	user := getEnvRequired("DB_USER")
	password := getEnvRequired("DB_PASSWORD")
	dbname := getEnvRequired("DB_NAME")
	port := getEnvRequired("DB_PORT")

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
	if err := DB.AutoMigrate(
		&models.Account{},
		&models.Group{},
		&models.GroupMember{},
		&models.ActivityLog{},
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

// WithTx executes function within a transaction
//not being used currently
// func WithTx(fn func(tx *gorm.DB) error) error {
// 	return DB.Transaction(fn)
// }
