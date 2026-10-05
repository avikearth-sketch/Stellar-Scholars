package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetAPODHandler(client *api.APODClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.APODParams

		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
			return
		}

		apodData, err := client.FetchAPOD(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch APOD data from NASA API",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(apodData),
			"data":    apodData,
		})
	}
}
