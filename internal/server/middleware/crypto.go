package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ASRafalsky/internal/server/authorization"
)

type ContextKey string

func WithAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) > 0 {
				tokenString = strings.TrimSuffix(strings.TrimPrefix(parts[0], "auth_token="), ";")
			}
		}
		if tokenString == "" {
			if cookie, err := r.Cookie("auth_token"); err == nil {
				tokenString = cookie.Value
			}
		}
		if tokenString == "" {
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		claims := &authorization.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return authorization.JWTKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKey("userLogin"), claims.Login)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}
