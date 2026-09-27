// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// BlacklistUseCase manages the vehicle blacklist.
//
// README lists blacklist enforcement as a core feature, but until now it existed
// only in documentation. Entries are managed here and enforced in the entry flow:
// a plate with an active blacklist record is denied entry with the gate closed.
type BlacklistUseCase struct {
	repo VehicleRepo
	log  *log.Helper
}

// NewBlacklistUseCase creates a new BlacklistUseCase.
func NewBlacklistUseCase(repo VehicleRepo, logger log.Logger) *BlacklistUseCase {
	return &BlacklistUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// AddBlacklistEntry adds (or re-activates) a blacklist record for a plate.
func (uc *BlacklistUseCase) AddBlacklistEntry(ctx context.Context, plateNumber, reason, createdBy string) (*BlacklistEntry, error) {
	plateNumber = strings.TrimSpace(plateNumber)
	if plateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	existing, err := uc.repo.GetBlacklistEntry(ctx, plateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing blacklist entry: %w", err)
	}

	if existing != nil {
		// Re-adding a previously removed plate re-activates it and refreshes the reason.
		if err := uc.repo.SetBlacklistEntryActive(ctx, plateNumber, true); err != nil {
			return nil, fmt.Errorf("failed to activate blacklist entry: %w", err)
		}
		existing.Active = true
		existing.Reason = reason
		existing.CreatedBy = createdBy
		return existing, nil
	}

	entry := &BlacklistEntry{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		Reason:      reason,
		CreatedBy:   createdBy,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := uc.repo.CreateBlacklistEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create blacklist entry: %w", err)
	}
	return entry, nil
}

// RemoveBlacklistEntry deactivates the blacklist record for a plate. Removal keeps
// the row (with active=false) so the enforcement history stays auditable.
func (uc *BlacklistUseCase) RemoveBlacklistEntry(ctx context.Context, plateNumber string) error {
	plateNumber = strings.TrimSpace(plateNumber)
	if plateNumber == "" {
		return fmt.Errorf("plate number is required")
	}

	entry, err := uc.repo.GetBlacklistEntry(ctx, plateNumber)
	if err != nil {
		return fmt.Errorf("failed to get blacklist entry: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("blacklist entry not found: %s", plateNumber)
	}

	return uc.repo.SetBlacklistEntryActive(ctx, plateNumber, false)
}

// ListBlacklistEntries lists blacklist records with pagination.
func (uc *BlacklistUseCase) ListBlacklistEntries(ctx context.Context, page, pageSize int) ([]*BlacklistEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.ListBlacklistEntries(ctx, page, pageSize)
}

// CheckBlacklist reports whether a plate is currently blacklisted.
func (uc *BlacklistUseCase) CheckBlacklist(ctx context.Context, plateNumber string) (*BlacklistEntry, error) {
	plateNumber = strings.TrimSpace(plateNumber)
	if plateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	entry, err := uc.repo.GetBlacklistEntry(ctx, plateNumber)
	if err != nil {
		return nil, err
	}
	if entry == nil || !entry.Active {
		return nil, nil
	}
	return entry, nil
}
