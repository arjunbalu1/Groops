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

	// Enable PostgreSQL extensions for advanced search
	if err := enableSearchExtensions(DB); err != nil {
		log.Printf("Warning: Failed to enable search extensions: %v", err)
	}

	if err := DB.AutoMigrate(
		&models.Account{},
		&models.Group{},
		&models.GroupMember{},
		&models.ActivityLog{},
		&models.Notification{},
		&models.Session{},
		&models.LoginLog{},
		&models.ReminderSent{},
		&models.Message{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Set up search indexes and triggers after migration
	if err := setupSearchIndexes(DB); err != nil {
		log.Printf("Warning: Failed to setup search indexes: %v", err)
	}

	log.Println("Database connection established and migrations completed")
	return nil
}

// enableSearchExtensions enables PostgreSQL extensions for advanced search
func enableSearchExtensions(db *gorm.DB) error {
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",  // Fuzzy matching
		"CREATE EXTENSION IF NOT EXISTS unaccent", // Remove accents for better search
	}

	for _, ext := range extensions {
		if err := db.Exec(ext).Error; err != nil {
			log.Printf("Failed to enable extension: %s, error: %v", ext, err)
			// Don't return error, just log warning
		}
	}

	return nil
}

// setupSearchIndexes creates indexes and triggers for full-text search
func setupSearchIndexes(db *gorm.DB) error {
	// Setup search extensions and indexes
	log.Println("Setting up search extensions and indexes...")

	// Enable required extensions
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		log.Printf("Warning: Failed to create pg_trgm extension: %v", err)
	}

	// Add search vector column
	if err := db.Exec(`
		ALTER TABLE "group" 
		ADD COLUMN IF NOT EXISTS search_vector tsvector
	`).Error; err != nil {
		log.Printf("Warning: Failed to add search_vector column: %v", err)
	}

	// Create search vector update function
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION update_group_search_vector() RETURNS trigger AS $$
		BEGIN
			NEW.search_vector := 
				setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
				setweight(to_tsvector('english', coalesce(NEW.activity_type, '')), 'A') ||
				setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B') ||
				setweight(to_tsvector('english', coalesce(NEW.organiser_id, '')), 'D');
			RETURN NEW;
		END
		$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create search vector function: %v", err)
	}

	// Drop existing trigger if exists
	if err := db.Exec(`DROP TRIGGER IF EXISTS group_search_vector_update ON "group"`).Error; err != nil {
		log.Printf("Warning: Failed to drop existing trigger: %v", err)
	}

	// Create trigger
	if err := db.Exec(`
		CREATE TRIGGER group_search_vector_update 
		BEFORE INSERT OR UPDATE ON "group" 
		FOR EACH ROW EXECUTE FUNCTION update_group_search_vector()
	`).Error; err != nil {
		log.Printf("Warning: Failed to create search vector trigger: %v", err)
	}

	// Create search indexes
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_search_vector ON "group" USING GIN (search_vector)`).Error; err != nil {
		log.Printf("Warning: Failed to create search vector index: %v", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_name_trgm ON "group" USING GIN (name gin_trgm_ops)`).Error; err != nil {
		log.Printf("Warning: Failed to create name trigram index: %v", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_activity_trgm ON "group" USING GIN (activity_type gin_trgm_ops)`).Error; err != nil {
		log.Printf("Warning: Failed to create activity trigram index: %v", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_description_trgm ON "group" USING GIN (description gin_trgm_ops)`).Error; err != nil {
		log.Printf("Warning: Failed to create description trigram index: %v", err)
	}

	// Update search vectors for existing records
	if err := db.Exec(`
		UPDATE "group" SET search_vector = 
			setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
			setweight(to_tsvector('english', coalesce(activity_type, '')), 'A') ||
			setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
			setweight(to_tsvector('english', coalesce(organiser_id, '')), 'D')
		WHERE search_vector IS NULL
	`).Error; err != nil {
		log.Printf("Warning: Failed to update existing search vectors: %v", err)
	}

	log.Println("Search setup completed")
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
