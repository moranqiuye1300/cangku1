package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"short-video-platform/user-service/internal/model"
)

func OpenMySQL() (*gorm.DB, error) {
	host := getenv("MYSQL_HOST", "127.0.0.1")
	port := getenv("MYSQL_PORT", "3307")
	user := getenv("MYSQL_USER", "svp")
	pass := getenv("MYSQL_PASSWORD", "svp123456")
	dbName := getenv("MYSQL_DATABASE", "short_video")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, dbName,
	)
	log.Printf("connecting mysql: %s@%s:%s/%s", user, host, port, dbName)

	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}
	var db *gorm.DB
	var err error

	for i := 1; i <= 30; i++ {
		db, err = gorm.Open(mysql.Open(dsn), cfg)
		if err != nil {
			log.Printf("mysql attempt %d/30: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}
		sqlDB, e := db.DB()
		if e != nil {
			err = e
			time.Sleep(2 * time.Second)
			continue
		}
		if err = sqlDB.Ping(); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(time.Hour)
		return db, nil
	}
	return nil, fmt.Errorf("mysql connect failed: %w", err)
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.UserRecord{})
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
