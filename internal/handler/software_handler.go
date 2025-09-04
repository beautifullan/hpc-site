package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
	"net/http"
	"strconv"
)

func GetSoftware(c *gin.Context) {
	ctx := context.Background()

	// 获取 query 参数
	name := c.Query("name")
	category := c.Query("category")
	tag := c.Query("tag")
	search := c.Query("search")

	softwares, err := repository.QuerySoftware(ctx, name, category, tag, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, softwares)
}

func GetSoftwareDetail(c *gin.Context) {
	ctx := context.Background()
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 查软件
	software, err := repository.GetSoftwareByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "software not found"})
		return
	}

	// 查相关论文
	papers, err := repository.GetPapersBySoftwareID(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"software": software,
		"papers":   papers,
	})
}
