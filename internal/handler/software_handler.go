package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
	"net/http"
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
