package server

import (
	"fmt"
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
	r := Routes()

	addr := os.Getenv("LANGEKKO_ADDR")
	port := os.Getenv("LANGEKKO_PORT")
	serverAddr := fmt.Sprintf("%s:%s", addr, port)

	log.Printf("Starting server on %s", serverAddr)
	err = http.ListenAndServe(serverAddr, r)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

// define mux routes
func Routes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/language", LanguageHandler)
	http.Handle("/", r)
	return r
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {

}

func LanguageHandler(w http.ResponseWriter, r *http.Request) {

}
