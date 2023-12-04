package oauth2

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

func TestCallbackHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/auth/google/callback", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(callbackHandler)

	handler.ServeHTTP(rr, req)

	// check it returns 307
	assert.Equal(t, http.StatusOK, rr.Code)
	// Add more assertions as needed
}

func TestBeginAuthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/auth/google", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(beginAuthHandler)

	handler.ServeHTTP(rr, req)

}

func TestLogoutHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/auth/logout", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(logoutHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	// Add more assertions as needed
}
