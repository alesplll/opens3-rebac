package utils

import "regexp"

// Helper function for email validation
func IsValidEmail(email string) bool {
	// Simple regex for basic email validation
	const emailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	return regexp.MustCompile(emailRegex).MatchString(email)
}
