package pkg

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq" // Postgres driver
)

var DB *sql.DB

func InitDB() {
	// 从环境变量读取 DATABASE_URL
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ DATABASE_URL 未设置")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	// 测试连接
	if err := DB.Ping(); err != nil {
		log.Fatalf("❌ 数据库不可用: %v", err)
	}

	log.Println("✅ 数据库连接成功")
}
