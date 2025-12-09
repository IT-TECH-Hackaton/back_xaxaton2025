package main

import (
	"log"

	"bekend/config"
	"bekend/database"
	"bekend/routes"
	"bekend/services"
)

func main() {
	config.LoadConfig()
	database.Connect()

	cronService := services.NewCronService()
	cronService.Start()

	r := routes.SetupRoutes()

	port := config.AppConfig.AppPort
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

