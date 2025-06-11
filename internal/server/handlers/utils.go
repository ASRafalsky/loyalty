package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"
)

func checkLogin(login string) bool {
	switch {
	case len(login) < 4:
		return false
	case len(login) > 100:
		return false
	default:
	}
	return true
}

func checkPassword(password string) bool {
	switch {
	case len(password) < 4:
		return false
	case len(password) > 100:
		return false
	default:
	}
	return true
}

func isValidOrderByLuhn(id string) bool {
	n := len(id)
	if n < 2 {
		return false
	}

	digits := []rune(id)
	total := 0
	double := false

	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
		digit := int(digits[i] - '0')

		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		total += digit
		double = !double
	}

	return total%10 == 0
}

func cleanSpaces(s string) string {
	cleaned := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			cleaned = append(cleaned, r)
		}
	}
	return string(cleaned)
}

func setToken(res http.ResponseWriter, req *http.Request, token string, expTime time.Time) error {
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

func CompareFloat64(a, b, epsilon float64) bool {
	return math.Abs(a-b) < 1e-10
}

func main() {
	a := 0.1 + 0.2
	b := 0.3
	epsilon := 1e-10 // Уровень точности

	fmt.Println(CompareFloat64(a, b, epsilon)) // true
}
