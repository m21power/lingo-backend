package main

import (
	"lingo-backend/routes"
	"log"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	route := mux.NewRouter()
	r := routes.NewRouter(route)

	// Register all your routes and start the bot in background
	r.RegisterRoute()

	// Bot will run in a goroutine from inside RegisterRoute()
	// (if you already did: go bot.ListenToBot(...) there)

	// Start HTTP server on the port Render needs
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Running on port", port)
	r.Run(":" + port)
}
