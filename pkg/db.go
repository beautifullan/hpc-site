package pkg

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB() {
	dsn := os.Getenv("DATABASE_URL") // 从环境变量读取
	if dsn == "" {
		log.Fatal("环境变量 DATABASE_URL 未设置")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 测试一下连接
	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("数据库无法 Ping 通: %v", err)
	}

	DB = pool
	fmt.Println("✅ 成功连接到数据库")
}
