package pkg

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("环境变量 DATABASE_URL 未设置")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 测试连接
	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("数据库无法 Ping 通: %v", err)
	}

	DB = pool
	log.Println("✅ 成功连接到数据库")
}
