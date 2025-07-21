package utils

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail checks if email format is valid
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > 254 {
		return errors.New("email is too long")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// ValidatePassword checks password strength
func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return errors.New("password is too long")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}

// ValidateUsername checks username format
func ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username is required")
	}
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if len(username) > 50 {
		return errors.New("username is too long")
	}

	// Allow alphanumeric, underscore, and hyphen
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// SanitizeString removes dangerous characters and trims whitespace
func SanitizeString(input string) string {
	// Remove null bytes and control characters
	sanitized := strings.ReplaceAll(input, "\x00", "")
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// ValidateAuctionTitle validates auction title
func ValidateAuctionTitle(title string) error {
	title = SanitizeString(title)
	if title == "" {
		return errors.New("auction title is required")
	}
	if len(title) < 5 {
		return errors.New("auction title must be at least 5 characters long")
	}
	if len(title) > 200 {
		return errors.New("auction title is too long")
	}
	return nil
}

// ValidateAuctionDescription validates auction description
func ValidateAuctionDescription(description string) error {
	description = SanitizeString(description)
	if description == "" {
		return errors.New("auction description is required")
	}
	if len(description) < 10 {
		return errors.New("auction description must be at least 10 characters long")
	}
	if len(description) > 2000 {
		return errors.New("auction description is too long")
	}
	return nil
}

// ValidatePrice validates monetary values
func ValidatePrice(price float64, fieldName string) error {
	if price <= 0 {
		return errors.New(fieldName + " must be positive")
	}
	if price > 10_000_000_000_000 {
		return errors.New(fieldName + " is too large")
	}
	return nil
}
