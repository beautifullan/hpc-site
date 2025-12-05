package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
	"log"
)

// GET /benchmark
func GetBenchmarks(c *gin.Context) {
	ctx := context.Background()
	benchmarks, err := repository.GetAllBenchmarks(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 benchmark 列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, benchmarks)
}

// GET /software/:id/benchmark
func GetBenchmarksBySoftware(c *gin.Context) {
	idStr := c.Param("id")
	softwareID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "软件 ID 无效"})
		return
	}

	ctx := context.Background()
	benchmarks, err := repository.GetBenchmarksBySoftwareID(ctx, softwareID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 benchmark 失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, benchmarks)
}

func GetBenchmarkResults(c *gin.Context) {
	benchmarkIDStr := c.Param("benchmarkID")
	benchmarkID, err := strconv.Atoi(benchmarkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Benchmark ID"})
		return
	}
	ctx := c.Request.Context()
	rawResults, err := repository.GetAllBechmarkResults(ctx, benchmarkID)

	// 注意：仓库方法名称应为 GetBenchmarkResultsByBenchmarkID，这里使用你的 GetAllBechmarkResults
	results, err := repository.GetAllBechmarkResults(c.Request.Context(), benchmarkID)
	if err != nil {
		log.Print(c.Request.Context(), "获取 Benchmark 运行结果失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Benchmark 运行结果失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
