package biz

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseAmount parses a WeChat APIv2 amount string expressed in cents and returns yuan.
//
// It is only kept for logging and legacy APIv2 reporting. Settlements compare integer
// cents (see parseYuanToCents) so that no money decision depends on binary floating point.
func parseAmount(amount string) float64 {
	cents, err := strconv.ParseInt(strings.TrimSpace(amount), 10, 64)
	if err != nil {
		return 0
	}
	return float64(cents) / 100
}

// parseAmountFloat parses a decimal amount string into float64.
//
// Prefer parseYuanToCents anywhere the value participates in a comparison or arithmetic.
func parseAmountFloat(amount string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return 0
	}
	return f
}

// parseYuanToCents converts a decimal yuan string such as "12.34" into integer cents.
//
// The digits are parsed directly rather than via ParseFloat: binary floating point cannot
// represent most decimal fractions exactly (0.1 + 0.2 != 0.3), so routing money through
// float64 accumulates rounding drift that shows up as reconciliation mismatches.
func parseYuanToCents(amount string) (int64, error) {
	s := strings.TrimSpace(amount)
	if s == "" {
		return 0, fmt.Errorf("amount is empty")
	}

	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = strings.TrimSpace(s[1:])
	}

	var yuanPart, centPart string
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		yuanPart = s[:idx]
		centPart = s[idx+1:]
	} else {
		yuanPart = s
	}
	if yuanPart == "" {
		yuanPart = "0"
	}
	if yuanPart == "" || !allDigits(yuanPart) || !allDigits(centPart) {
		return 0, fmt.Errorf("amount %q is not a decimal number", amount)
	}

	switch {
	case len(centPart) < 2:
		centPart += strings.Repeat("0", 2-len(centPart))
	case len(centPart) > 2:
		// Sub-cent precision is a contract violation, not something to round away silently.
		if !allZero(centPart[2:]) {
			return 0, fmt.Errorf("amount %q carries sub-cent precision", amount)
		}
		centPart = centPart[:2]
	}

	yuan, err := strconv.ParseInt(yuanPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q overflows: %w", amount, err)
	}
	cent, err := strconv.ParseInt(centPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q is malformed: %w", amount, err)
	}

	total := yuan*100 + cent
	if negative {
		total = -total
	}
	return total, nil
}

func allDigits(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func allZero(s string) bool {
	for _, r := range s {
		if r != '0' {
			return false
		}
	}
	return true
}

// currentTime returns current time.
func currentTime() time.Time {
	return time.Now()
}
