package repository

import (
	"context"
	"fmt"
	"hpc-site/pkg"
	"testing"
)

func TestGetAllSoftware(t *testing.T) {
	// 初始化数据库连接
	dsn := "postgres://admin:admin@localhost:5432/hpcdb?sslmode=disable"
	pkg.InitDB(dsn)

	// 调用查询
	softwareList, err := GetAllSoftware(context.Background())
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	// 打印结果
	for _, s := range softwareList {
		fmt.Printf("ID=%d, Name=%s, Tags=%v, Categories=%v\n", s.ID, s.Name, s.Tags, s.Categories)
	}
}
