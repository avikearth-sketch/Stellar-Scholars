package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetGIBSTileURLHandler(client *api.GIBSClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.GIBSTileParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
			return
		}

		tileInfo := client.BuildTileURL(params)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    tileInfo,
		})
	}
}

func GetGIBSCapabilitiesHandler(client *api.GIBSClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		projection := c.DefaultQuery("projection", "epsg4326")

		xmlContent, err := client.FetchCapabilities(c.Request.Context(), projection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch GIBS WMTS Capabilities",
				"details": err.Error(),
			})
			return
		}

		c.Data(http.StatusOK, "application/xml", []byte(xmlContent))
	}
}
