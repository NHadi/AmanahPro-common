package persistence

import (
	"log"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// InitializeDB initializes the database connection using GORM and returns the DB instance.
func InitializeDB(DATABASE_URL string) (*gorm.DB, error) {
	// Open database connection
	db, err := gorm.Open(sqlserver.Open(DATABASE_URL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
		return nil, err
	}

	return db, nil
}
