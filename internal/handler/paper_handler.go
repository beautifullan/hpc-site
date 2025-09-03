package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"hpc-site/internal/repository"
)

// GET /papers
func GetPapers(c *gin.Context) {
	papers, err := repository.GetAllPapers(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, papers)
}
