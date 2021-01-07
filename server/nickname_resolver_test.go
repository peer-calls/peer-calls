package server_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
)

func sampleToken() string {
	// Create the token
	token := jwt.New(jwt.GetSigningMethod("HS256"))

	token.Claims = jwt.MapClaims{
		"iat":   time.Now().Unix(),
		"email": "test@mail.service.co",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	tokenString, _ := token.SignedString([]byte("secret"))

	jsonb, _ := json.Marshal(struct {
		Token string `json:"token"`
	}{tokenString})

	return string(jsonb)
}

func setSampleCookie(r *http.Request) {
	cookie := &http.Cookie{
		Name:   "auth",
		Value:  url.QueryEscape(sampleToken()),
		MaxAge: int(2 * time.Hour.Seconds()),
		Path:   "/",
		Domain: "",
	}
	r.Header.Set("Cookie", cookie.String())
}

func TestParseClaims(t *testing.T) {
	var cookieValue = sampleToken()
	v, _ := url.QueryUnescape(cookieValue)
	nr := server.NewJwtNicknameResolver(server.JwtHeaders{
		CookieTokenName: "token",
		NicknameClaim:   "email",
	})
	nickname := nr.ParseClaims(v)
	require.Equal(t, "test@mail.service.co", nickname)
}
