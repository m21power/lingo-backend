package routes

import (
	"lingo-backend/db"
	"log"
	"net/http"

	bot "lingo-backend/controllers"

	handlers "lingo-backend/controllers/handlers"
	repository "lingo-backend/controllers/repository"
	usecases "lingo-backend/usecase"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Router struct {
	route *mux.Router
}

func NewRouter(r *mux.Router) *Router {
	return &Router{route: r}
}

func (r *Router) RegisterRoute() {
	// Connect to DB
	database, err := db.ConnectDb()
	if err != nil {
		log.Println("Cannot connect to db")
		return
	}

	log.Println("Routes registered:")
	go bot.ListenToBot(database) // Start listening to the Telegram bot

	// Student endpoint
	otpRepository := repository.NewOtpRepository(database)
	otpUsecase := usecases.NewOtpUsecase(otpRepository)
	otpHandler := handlers.NewOtpHandler(*otpUsecase)

	// Define route prefix
	otpRoutes := r.route.PathPrefix("/api/v1").Subrouter()
	otpRoutes.HandleFunc("/otp", otpHandler.CheckOtp).Methods("POST")
}

func (r *Router) Run(addr string) error {

	// CORS configuration to allow all origins
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                                       // Allow all origins
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // Allowed methods
		AllowedHeaders: []string{"Content-Type", "Authorization"},           // Allowed headers
	})

	// Wrap the mux router with CORS middleware
	handler := corsHandler.Handler(r.route)

	// Run the server with CORS enabled
	log.Println("Server running on port: ", addr)
	return http.ListenAndServe(addr, handler)
}
