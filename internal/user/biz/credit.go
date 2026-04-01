package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// CreditManager handles credit-related business logic
type CreditManager struct {
	userRepo UserRepo
	log      *log.Helper
}

// NewCreditManager creates a new CreditManager
func NewCreditManager(userRepo UserRepo, logger log.Logger) *CreditManager {
	return &CreditManager{
		userRepo: userRepo,
		log:      log.NewHelper(logger),
	}
}

// CalculateCreditScore calculates the user's credit score based on their payment history
func (cm *CreditManager) CalculateCreditScore(ctx context.Context, userID string) (int, error) {
	user, err := cm.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Get user's payment history and calculate score
	// For simplicity, we'll use a basic algorithm here
	// In a real system, you would consider factors like:
	// - Payment timeliness
	// - Number of late payments
	// - Total amount spent
	// - Frequency of use
	
	// Get payment records (this would be implemented in a real system)
	paymentRecords, err := cm.userRepo.GetUserPaymentRecords(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Calculate score based on payment records
	score := 100 // Base score
	latePayments := 0
	totalPayments := len(paymentRecords)

	for _, record := range paymentRecords {
		if record.IsLate {
			latePayments++
			score -= 10 // Deduct 10 points for each late payment
		}
	}

	// Add points for consistent payments
	if totalPayments > 0 {
		onTimeRate := float64(totalPayments-latePayments) / float64(totalPayments)
		if onTimeRate >= 0.9 {
			score += 10 // Add 10 points for good payment history
		} else if onTimeRate >= 0.7 {
			score += 5 // Add 5 points for average payment history
		}
	}

	// Ensure score is within valid range
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	// Update user's credit score
	user.CreditScore = score
	user.CreditLevel = cm.CalculateCreditLevel(score)
	user.UpdatedAt = time.Now()

	if err := cm.userRepo.UpdateUser(ctx, user); err != nil {
		return 0, err
	}

	return score, nil
}

// CalculateCreditLevel calculates the user's credit level based on their score
func (cm *CreditManager) CalculateCreditLevel(score int) string {
	switch {
	case score >= 90:
		return "A+"
	case score >= 80:
		return "A"
	case score >= 70:
		return "B+"
	case score >= 60:
		return "B"
	case score >= 50:
		return "C"
	default:
		return "D"
	}
}

// CheckCreditEligibility checks if user is eligible for post-payment based on credit score
func (cm *CreditManager) CheckCreditEligibility(ctx context.Context, userID string) (bool, error) {
	user, err := cm.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Users with credit score >= 60 are eligible for post-payment
	return user.CreditScore >= 60, nil
}

// UpdateCreditScore updates the user's credit score based on a payment event
func (cm *CreditManager) UpdateCreditScore(ctx context.Context, userID string, isLate bool) error {
	// Calculate and update credit score
	_, err := cm.CalculateCreditScore(ctx, userID)
	return err
}
