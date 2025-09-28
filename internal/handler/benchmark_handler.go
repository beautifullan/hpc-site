package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
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
