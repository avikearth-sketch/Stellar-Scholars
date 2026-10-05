package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetEPICRecentHandler(client *api.EPICClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		imageType := c.DefaultQuery("type", "natural")

		images, err := client.FetchRecent(c.Request.Context(), imageType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch EPIC imagery",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(images),
			"data":    images,
		})
	}
}

func GetEPICByDateHandler(client *api.EPICClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Param("date")
		imageType := c.DefaultQuery("type", "natural")

		if date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Date parameter is required (YYYY-MM-DD)"})
			return
		}

		images, err := client.FetchByDate(c.Request.Context(), imageType, date)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch EPIC imagery for date",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(images),
			"data":    images,
		})
	}
}

func GetEPICAllDatesHandler(client *api.EPICClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		imageType := c.DefaultQuery("type", "natural")

		dates, err := client.FetchAvailableDates(c.Request.Context(), imageType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch available EPIC dates",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(dates),
			"data":    dates,
		})
	}
}
