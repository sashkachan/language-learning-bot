package user

import (
	"net/http"

	"github.com/gorilla/mux"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type UserSettings struct {
	UserID      int64   `json:"user_id"`
	Language    string  `json:"language"`
	HelpType    string  `json:"help_type"`
	Level       string  `json:"level"`
	Translation string  `json:"translation"`
	SpeechSpeed float64 `json:"speech_speed"`
}

type UserRegistration struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	// allow registering without a password through gmail, facebook, etc.
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/register", registerHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.HandleFunc("/user", getUserInfoHandler).Methods("GET")
	r.HandleFunc("/user", updateUserInfoHandler).Methods("PUT")
	r.HandleFunc("/user", deleteUserInfoHandler).Methods("DELETE")
	r.HandleFunc("/user/settings", getUserSettingsHandler).Methods("GET")
	return r
}

// registration handler
func registerHandler(w http.ResponseWriter, r *http.Request) {

}

// login handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement login logic
}

// logout handler
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout logic
}

// get user info handler
func getUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get user info logic
}

// update user info handler
func updateUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update user info logic
}

// delete user handler
func deleteUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement delete user logic
}

// get user settings handler
func getUserSettingsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get user settings logic
}
