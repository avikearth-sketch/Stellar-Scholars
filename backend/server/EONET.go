package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetEONETEventsHandler(client *api.EONETClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.EONETEventParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
			return
		}

		eventsData, err := client.FetchEvents(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch EONET events from NASA",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(eventsData.Events),
			"data":    eventsData,
		})
	}
}

func GetEONETCategoriesHandler(client *api.EONETClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoriesData, err := client.FetchCategories(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch EONET categories from NASA",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   len(categoriesData.Categories),
			"data":    categoriesData.Categories,
		})
	}
}
