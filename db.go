package main

import (
	"sync"

	"github.com/TXOne-Stellar/stellar-lib/logging"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var dbInitOnce sync.Once

func DB() *gorm.DB {
	dbInitOnce.Do(func() {
		var err error
		dsn := "host=localhost port=5432 user=user password=pass sslmode=disable"
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			logging.Fatal("db error: %s", err)
		}
	})

	return db
}

func DryRun(db *gorm.DB) *gorm.DB {
	return db.Session(&gorm.Session{DryRun: true})
}
