package oauth2

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	goth "github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	google "github.com/markbates/goth/providers/google"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/{provider}", beginAuthHandler)
	router.HandleFunc("/logout/{provider}", logoutHandler)
	router.HandleFunc("/{provider}/callback", callbackHandler)

	return router
}

// other imports

func init() {
	// use godotenv to load env variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
	}

	scheme := os.Getenv("LANGEKKO_SCHEME")
	addr := os.Getenv("LANGEKKO_ADDR")
	port := os.Getenv("LANGEKKO_PORT")
	gothic.Store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_OAUTH2_CLIENT_ID"), os.Getenv("GOOGLE_OAUTH2_SECRET"), fmt.Sprintf("%s://%s:%s/auth/google/callback", scheme, addr, port), "email", "profile"),
	)

	log.Printf("Google OAuth2 callback URL: %s://%s:%s/auth/google/callback\n", scheme, addr, port)
}

func beginAuthHandler(w http.ResponseWriter, r *http.Request) {
	// try to get the user without re-authenticating
	if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
		t, _ := template.New("foo").Parse(userTemplate)
		t.Execute(w, gothUser)
	} else {
		gothic.BeginAuthHandler(w, r)
	}
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	t, _ := template.New("foo").Parse(userTemplate)
	t.Execute(w, user)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	gothic.Logout(w, r)
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

var userTemplate = `
<p><a href="/auth/logout/{{.Provider}}">logout</a></p>
<p>Name: {{.Name}} [{{.LastName}}, {{.FirstName}}]</p>
<p>Email: {{.Email}}</p>
<p>NickName: {{.NickName}}</p>
<p>Location: {{.Location}}</p>
<p>AvatarURL: {{.AvatarURL}} <img src="{{.AvatarURL}}"></p>
<p>Description: {{.Description}}</p>
<p>UserID: {{.UserID}}</p>
<p>AccessToken: {{.AccessToken}}</p>
<p>ExpiresAt: {{.ExpiresAt}}</p>
<p>RefreshToken: {{.RefreshToken}}</p>
`
