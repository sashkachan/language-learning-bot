package server

import (
	"fmt"
	auth "language-learning-bot/pkg/oauth2"
	"language-learning-bot/pkg/user"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func StartServer() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
	}
	r := apiv1Routes()

	// scheme := os.Getenv("LANGEKKO_SCHEME")
	addr := os.Getenv("LANGEKKO_ADDR")
	port := os.Getenv("LANGEKKO_PORT")
	serverAddr := fmt.Sprintf("%s:%s", addr, port)

	log.Printf("Starting server on %s", serverAddr)
	err = http.ListenAndServe(serverAddr, r)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// define mux routes
func apiv1Routes() *mux.Router {
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	v1 := r.PathPrefix("/api/v1").Subrouter()
	userRouter := user.NewRouter()
	authRouter := auth.NewRouter()
	v1.PathPrefix("/user").Handler(http.StripPrefix("/api/v1/user", userRouter))
	v1.PathPrefix("/auth").Handler(http.StripPrefix("/api/v1/auth", authRouter))
	http.Handle("/", r)
	return r
}
