package config

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	model "task-pool-system.com/task-pool-system/internal/models"
)

func New(dsn string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}

	if err := db.AutoMigrate(&model.Task{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	return db
}
