package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetCMEHandler(client *api.DONKIClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.BaseDONKIParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.FetchCME(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch CME data from DONKI", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "count": len(data), "data": data})
	}
}

func GetSolarFlaresHandler(client *api.DONKIClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.BaseDONKIParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.FetchSolarFlares(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Solar Flare data from DONKI", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "count": len(data), "data": data})
	}
}

func GetGeomagneticStormsHandler(client *api.DONKIClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.BaseDONKIParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.FetchGeomagneticStorms(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch GST data from DONKI", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "count": len(data), "data": data})
	}
}

func GetNotificationsHandler(client *api.DONKIClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.NotificationParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.FetchNotifications(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch DONKI notifications", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "count": len(data), "data": data})
	}
}
