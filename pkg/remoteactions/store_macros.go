package remoteactions

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// MacroStore is the persistence layer for RemoteActionMacro rows.
// One instance per Service; safe for concurrent use.
type MacroStore struct {
	db *gorm.DB
}

func NewMacroStore(db *gorm.DB) *MacroStore { return &MacroStore{db: db} }

// Create inserts a new macro, stamping CreatedAt and UpdatedAt.
// TargetCall must already be uppercased — validation lives in the
// caller (validate.go).
func (s *MacroStore) Create(ctx context.Context, m *RemoteActionMacro) error {
	if m == nil {
		return errors.New("remoteactions: nil macro")
	}
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	return s.db.WithContext(ctx).Create(m).Error
}

// Get fetches by primary key. Returns gorm.ErrRecordNotFound when missing.
func (s *MacroStore) Get(ctx context.Context, id uint) (*RemoteActionMacro, error) {
	var m RemoteActionMacro
	if err := s.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// ListByTarget returns macros for one peer, ordered by Position
// ascending then ID ascending (tie-break for ties; defensive against
// reorder races).
func (s *MacroStore) ListByTarget(ctx context.Context, targetCall string) ([]RemoteActionMacro, error) {
	var out []RemoteActionMacro
	if err := s.db.WithContext(ctx).
		Where("target_call = ?", targetCall).
		Order("position ASC, id ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Update writes Label, ActionName, ArgsString, RemoteOTPCredentialID,
// Position, and bumps UpdatedAt. TargetCall is intentionally NOT
// updatable — the drawer is per-thread; moving a macro between targets
// is a delete+create.
func (s *MacroStore) Update(ctx context.Context, m *RemoteActionMacro) error {
	if m == nil || m.ID == 0 {
		return errors.New("remoteactions: nil macro or zero id")
	}
	m.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&RemoteActionMacro{}).
		Where("id = ?", m.ID).
		Updates(map[string]any{
			"label":                    m.Label,
			"action_name":              m.ActionName,
			"args_string":              m.ArgsString,
			"remote_otp_credential_id": m.RemoteOTPCredentialID,
			"position":                 m.Position,
			"updated_at":               m.UpdatedAt,
		}).Error
}

// Delete removes one macro by id.
func (s *MacroStore) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&RemoteActionMacro{}, id).Error
}

// Reorder rewrites the Position column of every macro for one
// targetCall to match the supplied id order (index 0 -> position 0,
// index 1 -> position 1, ...). Runs in a single transaction so partial
// failure leaves the existing order intact. Macros for the target that
// are NOT in `ids` are left alone — defensive against a stale UI list
// that lost a macro the operator deleted on another tab.
func (s *MacroStore) Reorder(ctx context.Context, targetCall string, ids []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		for i, id := range ids {
			res := tx.Model(&RemoteActionMacro{}).
				Where("id = ? AND target_call = ?", id, targetCall).
				Updates(map[string]any{
					"position":   i,
					"updated_at": now,
				})
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}
