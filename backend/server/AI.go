package server

import (
	"log"
	"net/http"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func AIChatHandler(service *api.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.AIChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		resp, err := service.Chat(c.Request.Context(), req)
		if err != nil {
			log.Printf("AI chat error: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"error":   "The AI service failed to produce an answer",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"configured": resp.Configured,
			"provider":   resp.Provider,
			"reply":      resp.Reply,
		})
	}
}
