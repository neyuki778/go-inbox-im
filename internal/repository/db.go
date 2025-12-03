package repository

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB 使用环境变量 IM_MYSQL_DSN 初始化 GORM 数据库连接。
// 默认值指向 docker-compose 中的本地 MySQL。
func NewDB() (*gorm.DB, error) {
	dsn := os.Getenv("IM_MYSQL_DSN")
	if dsn == "" {
		dsn = "im_user:im_pass123@tcp(localhost:8848)/go_im?parseTime=true&charset=utf8mb4&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// 简单的连接池配置，可按需调整
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Printf("已连接 MySQL: %s", dsn)
	return db, nil
}
