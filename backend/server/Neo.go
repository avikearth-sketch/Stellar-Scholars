package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetNeoFeedHandler(client *api.NeoWsClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.NeoFeedParams

		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
			return
		}

		feedData, err := client.FetchFeed(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch NeoWs feed from NASA",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"element_count": feedData.ElementCount,
			"data":          feedData.NearEarthObjects,
		})
	}
}

func GetNeoByIDHandler(client *api.NeoWsClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		asteroidID := c.Param("id")
		apiKey := c.Query("api_key")

		if asteroidID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Asteroid ID is required"})
			return
		}

		asteroid, err := client.FetchAsteroidByID(c.Request.Context(), asteroidID, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch asteroid details from NASA",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    asteroid,
		})
	}
}

func SetupRouter(neoClient *api.NeoWsClient) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1/neo")
	{
		api.GET("/feed", GetNeoFeedHandler(neoClient))
		api.GET("/:id", GetNeoByIDHandler(neoClient))
	}

	return r
}
