package server

import (
	"log"
	"net/http"
	"strconv"

	"backend/api"

	"github.com/gin-gonic/gin"
)

func GetRegionalTemperatureHandler(client *api.ClimateClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		vals := map[string]float64{}
		for _, name := range []string{"south", "north", "west", "east"} {
			v, err := strconv.ParseFloat(c.Query(name), 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid or missing query parameter: " + name,
				})
				return
			}
			vals[name] = v
		}

		data, err := client.FetchRegionalTemperature(c.Request.Context(), vals["south"], vals["north"], vals["west"], vals["east"])
		if err != nil {
			log.Printf("climate error: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"error":   "Failed to fetch regional temperature history",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
	}
}
