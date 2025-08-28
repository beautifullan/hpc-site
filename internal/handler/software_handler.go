package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
)

func GetSoftware(c *gin.Context) {
	software, err := repository.GetAllSoftware(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, software)
}
