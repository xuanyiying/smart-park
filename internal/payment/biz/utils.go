package biz

import (
	"strconv"
	"time"
)

// parseAmount parses amount string to float64 (converts cents to yuan).
func parseAmount(amount string) float64 {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0
	}
	return f / 100 // Convert cents to yuan
}

// parseAmountFloat parses amount string to float64.
func parseAmountFloat(amount string) float64 {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0
	}
	return f
}

// currentTime returns current time.
func currentTime() time.Time {
	return time.Now()
}
