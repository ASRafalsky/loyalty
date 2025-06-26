package authorization

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

var JWTKey = []byte("secret_key")

func NewToken(login string, exp time.Time) (string, error) {
	claims := &Claims{
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTKey)
}
