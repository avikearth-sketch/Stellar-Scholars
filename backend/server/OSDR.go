package server

import (
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetOSDRStudyFilesHandler(client *api.OSDRClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids := c.Param("ids")
		var params api.OSDRFilesParams

		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.FetchStudyFiles(c.Request.Context(), ids, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OSDR study files", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}

func GetOSDRStudyMetaHandler(client *api.OSDRClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		data, err := client.FetchStudyMetadata(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OSDR study metadata", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}

func SearchOSDRDatasetsHandler(client *api.OSDRClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params api.OSDRSearchParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
			return
		}

		data, err := client.SearchDatasets(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute OSDR dataset search", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}

func GetOSDREntityAllHandler(client *api.OSDRClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		entityType := c.Param("type")

		data, err := client.FetchEntityAll(c.Request.Context(), entityType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OSDR entity list", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}

func GetOSDREntitySingleHandler(client *api.OSDRClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		entityTypeSingular := c.Param("type")
		id := c.Param("id")

		data, err := client.FetchEntitySingle(c.Request.Context(), entityTypeSingular, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OSDR single entity detail", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}
