package main

import (
	"hpc-site/internal/handler"
	"hpc-site/pkg"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载.env 文件
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ .env 文件未找到，尝试使用系统环境变量")
	}

	// 初始化数据库（现在是 database/sql）
	pkg.InitDB()

	r := gin.Default()

	// 路由
	r.GET("/softwares", handler.GetSoftware)
	r.GET("/softwares/:id", handler.GetSoftwareDetail)
	r.GET("/papers", handler.GetPapers)
	// benchmark
	r.GET("/benchmarks", handler.GetBenchmarks)
	r.GET("/softwares/:id/benchmark", handler.GetBenchmarksBySoftware)
	r.POST("/crawl/all", handler.GetAllSoftwarePaper)
	r.POST("/test/single", handler.TestSinglePaper)

	log.Println("🚀 服务器启动: http://localhost:8080")
	r.Run(":8080")
}
