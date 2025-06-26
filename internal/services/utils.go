package services

import (
	"unicode"
)

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
