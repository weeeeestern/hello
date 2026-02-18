package main

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var err error

func init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		fmt.Printf("[%d/%d] Failed to connect to DB: %v\n", i, maxRetries, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		panic(fmt.Sprintf("Could not connect to DB after %d attempts: %v", maxRetries, err))
	}

	err = db.AutoMigrate(&Todo{})
	if err != nil {
		panic("AutoMigrate failed: " + err.Error())
	}
}