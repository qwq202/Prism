package auth

import (
	"net/mail"
	"regexp"
	"strings"
)

func isInRange(content string, min, max int) bool {
	content = strings.TrimSpace(content)
	return len(content) >= min && len(content) <= max
}

func validateUsername(username string) bool {
	return isInRange(username, 2, 24)
}

func validateUsernameOrEmail(username string) bool {
	return isInRange(username, 1, 255)
}

func validatePassword(password string) bool {
	return isInRange(password, 6, 36)
}

func ValidateEmailAddress(email string) bool {
	email = strings.TrimSpace(email)
	if !isInRange(email, 1, 255) {
		return false
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return false
	}

	domain := strings.TrimSuffix(strings.ToLower(parts[1]), ".")
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	labelExp := regexp.MustCompile(`^[a-z0-9-]+$`)
	tldExp := regexp.MustCompile(`^[a-z]{2,}$`)
	for _, label := range labels {
		if label == "" ||
			strings.HasPrefix(label, "-") ||
			strings.HasSuffix(label, "-") ||
			!labelExp.MatchString(label) {
			return false
		}
	}

	return tldExp.MatchString(labels[len(labels)-1])
}

func validateEmail(email string) bool {
	return ValidateEmailAddress(email)
}

func validateCode(code string) bool {
	return isInRange(code, 1, 64)
}
