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

	// 初始化数据库
	pkg.InitDB()

	r := gin.Default()

	r.GET("/software", handler.GetSoftware)
	r.GET("/paper", handler.GetPapers)

	r.Run(":8080")
}
