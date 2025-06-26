package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ASRafalsky/internal/server/authorization"
)

func setToken(res http.ResponseWriter, req *http.Request, login string, expTime time.Time) error {
	token, err := authorization.NewToken(login, expTime)
	if err != nil {
		return err
	}

	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		res.Header().Set("Content-Type", "application/json")
		res.Header().Set("Authorization", token)
		if err := json.NewEncoder(res).Encode(map[string]string{
			"token":   token,
			"expires": expTime.Format(time.RFC3339),
		}); err != nil {
			return err
		}
		return nil
	}
	http.SetCookie(res, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  expTime,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	return nil
}
