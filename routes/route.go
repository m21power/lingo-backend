package routes

import (
	"context"
	"lingo-backend/db"
	"log"
	"net/http"

	bot "lingo-backend/controllers"

	handlers "lingo-backend/controllers/handlers"
	repository "lingo-backend/controllers/repository"
	usecases "lingo-backend/usecase"

	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"google.golang.org/api/option"
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
	// ctx := context.Background()
	// opt := option.WithCredentialsFile("lingo-firestore.json")

	// app, err := firebase.NewApp(ctx, nil, opt)

	// if err != nil {
	// 	log.Println("Cannot connect to firestore")
	// 	// return
	// }
	ctx := context.Background()

	config := &firebase.Config{
		DatabaseURL: "https://lingo-19e2a-default-rtdb.firebaseio.com/",
	}

	app, err := firebase.NewApp(ctx, config, option.WithCredentialsFile("lingo-firestore.json"))
	if err != nil {
		log.Fatalf("Failed to initialize Firebase app: %v", err)
	}

	rtdbClient, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Realtime DB: %v", err)
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Println("Cannot connect to firestore")
		// return
	}

	// otp endpoint
	otpRepository := repository.NewOtpRepository(database, client)
	otpUsecase := usecases.NewOtpUsecase(otpRepository)
	otpHandler := handlers.NewOtpHandler(*otpUsecase)

	// Define route prefix
	routes := r.route.PathPrefix("/api/v1").Subrouter()

	routes.HandleFunc("/otp", otpHandler.CheckOtp).Methods("POST")
	routes.HandleFunc("/otp/wake-up", otpHandler.WakeUpRender).Methods("GET")
	// pair endpoint
	pairRepository := repository.NewPairRepository(database)
	pairUsecase := usecases.NewPairUsecase(pairRepository)
	pairHandler := handlers.NewPairHandler(*pairUsecase)

	// Define route prefix
	routes.HandleFunc("/pair/{userId}", pairHandler.GetDailyPairs).Methods("GET")
	routes.HandleFunc("/pair", pairHandler.UpdatePairParticipation).Methods("PUT")

	// user endpoint
	userRepository := repository.NewUserRepo(database, client, rtdbClient)
	userUsecase := usecases.NewUserUsecase(userRepository)
	userHandler := handlers.NewUserHandler(*userUsecase)

	// routes.HandleFunc("/ws", userHandler.HandleWebSocket)

	routes.HandleFunc("/user/attendance", userHandler.FillAttendance).Methods("POST")
	routes.HandleFunc("/user/pair", userHandler.PairUser).Methods("POST")
	routes.HandleFunc("/user/notifications/{userId}", userHandler.GetNotifications).Methods("GET")
	routes.HandleFunc("/user/seen-notification/{userId}", userHandler.SeenNotification).Methods("POST")
	routes.HandleFunc("/user/generate-pair", userHandler.GeneratePair).Methods("POST")

	log.Println("Routes registered:")
	go bot.ListenToBot(database, client)

	if err != nil {
		log.Println("Error generating daily pairs:", err)
		// return
	}

}

func (r *Router) Run(addr string) error {

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	handler := corsHandler.Handler(r.route)

	log.Println("Server running on port: ", addr)
	return http.ListenAndServe(addr, handler)
}
