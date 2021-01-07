package server

import (
	"net/http"
	"net/url"

	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fastjson"
)

type JwtNicknameResolver struct {
	JwtHeaders
}

type NicknameResolver interface {
	Nickname(*http.Request) (string, bool)
	ParseClaims(string) string
}

func NewJwtNicknameResolver(config JwtHeaders) NicknameResolver {
	return &JwtNicknameResolver{
		config,
	}
}

func (c *JwtNicknameResolver) Nickname(r *http.Request) (nickname string, ok bool) {
	if cookie, err := r.Cookie(c.CookieName); err == nil && cookie.Value != "" {
		cookieValue, _ := url.QueryUnescape(cookie.Value)
		nickname = c.ParseClaims(cookieValue)
		if nickname != "" {
			ok = true
			return
		}
	}

	return
}

func (c *JwtNicknameResolver) ParseClaims(cookieValue string) (nickname string) {
	json, err := fastjson.Parse(cookieValue)
	if err != nil {
		return
	}
	token := json.GetStringBytes(c.CookieTokenName)

	claims := jwt.MapClaims{}
	_, _, err = new(jwt.Parser).ParseUnverified(string(token), &claims)
	if err != nil {
		return
	}

	if v, ok := claims[c.NicknameClaim].(string); ok {
		nickname = v
	}

	return
}
