package main

import (
	"log"
	"net/http"
	"os"

	"backend/api"
	"backend/server"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("template/*")
	r.Static("/assets", "./assets")
	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{"title": "Be an earth system trend detective"})
	})

	nasaKey := os.Getenv("NASA_API_KEY")
	if nasaKey == "" {
		nasaKey = "DEMO_KEY"
	}

	apodClient := api.NewAPODClient(nasaKey)
	neoClient := api.NewNeoWsClient(nasaKey)
	donkiClient := api.NewDONKIClient(nasaKey)
	eonetClient := api.NewEONETClient()
	epicClient := api.NewEPICClient(nasaKey)
	gibsClient := api.NewGIBSClient()
	osdrClient := api.NewOSDRClient()
	climateClient := api.NewClimateClient()
	aiService := api.NewAIService()

	if aiService.Configured() {
		log.Printf("AI service: using %s", aiService.ProviderName())
	} else {
		log.Println("AI service: NOT configured (set GEMINI_API_KEY to enable)")
	}

	r.GET("/api/apod", server.GetAPODHandler(apodClient))

	neo := r.Group("/api/neo")
	{
		neo.GET("/feed", server.GetNeoFeedHandler(neoClient))
		neo.GET("/:id", server.GetNeoByIDHandler(neoClient))
	}

	donki := r.Group("/api/donki")
	{
		donki.GET("/cme", server.GetCMEHandler(donkiClient))
		donki.GET("/flr", server.GetSolarFlaresHandler(donkiClient))
		donki.GET("/gst", server.GetGeomagneticStormsHandler(donkiClient))
		donki.GET("/notifications", server.GetNotificationsHandler(donkiClient))
	}

	eonet := r.Group("/api/eonet")
	{
		eonet.GET("/events", server.GetEONETEventsHandler(eonetClient))
		eonet.GET("/categories", server.GetEONETCategoriesHandler(eonetClient))
	}

	epic := r.Group("/api/epic")
	{
		epic.GET("/recent", server.GetEPICRecentHandler(epicClient))
		epic.GET("/date/:date", server.GetEPICByDateHandler(epicClient))
		epic.GET("/dates", server.GetEPICAllDatesHandler(epicClient))
	}

	gibs := r.Group("/api/gibs")
	{
		gibs.GET("/tile", server.GetGIBSTileURLHandler(gibsClient))
		gibs.GET("/capabilities", server.GetGIBSCapabilitiesHandler(gibsClient))
	}

	osdr := r.Group("/api/osdr")
	{
		osdr.GET("/files/:ids", server.GetOSDRStudyFilesHandler(osdrClient))
		osdr.GET("/meta/:id", server.GetOSDRStudyMetaHandler(osdrClient))
		osdr.GET("/search", server.SearchOSDRDatasetsHandler(osdrClient))
		osdr.GET("/entities/:type", server.GetOSDREntityAllHandler(osdrClient))
		osdr.GET("/entity/:type/:id", server.GetOSDREntitySingleHandler(osdrClient))
	}

	r.GET("/api/climate/temperature", server.GetRegionalTemperatureHandler(climateClient))

	ai := r.Group("/api/ai")
	{
		ai.POST("/chat", server.AIChatHandler(aiService))
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
