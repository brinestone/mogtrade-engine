package kyc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// DefaultValidWindow is used when a submit request omits a validity window.
	DefaultValidWindow = 365 * 24 * time.Hour
	// ClaimStatusAbsent is the JWT/kyc claim value when the user has no KYC row.
	ClaimStatusAbsent = "absent"
)

var (
	ErrKycNotFound          = errors.New("kyc record not found")
	ErrKycAlreadyExists     = errors.New("kyc record already exists for user")
	ErrInvalidRiskProfile   = errors.New("invalid kyc risk profile")
	ErrInvalidStatus        = errors.New("invalid kyc status")
	ErrEmptyRejectionReason = errors.New("rejection reason is required")
	ErrEmptyUserID          = errors.New("user id is required")
)

type CreateParams struct {
	ID                 string
	UserID             string
	ValidWindow        time.Duration
	VerificationWindow time.Duration
}

type UpdateDocumentsParams struct {
	UserID         string
	IdentityDoc    contract.IdentityDocument
	ProofOfAddress contract.ProofOfAddressDocument
}

type VerifyParams struct {
	UserID      string
	VerifiedBy  string
	RiskProfile contract.KYCRiskProfile
}

type RejectParams struct {
	UserID          string
	VerifiedBy      string
	RejectionReason string
}

type SetStatusParams struct {
	UserID string
	Status contract.KYCStatus
}

// CreateRecord inserts a pending KYC application for the user.
func CreateRecord(ctx context.Context, q *db.Queries, p CreateParams) (db.LookupActiveKycForUserRow, error) {
	if p.UserID == "" {
		return db.LookupActiveKycForUserRow{}, ErrEmptyUserID
	}
	window := p.ValidWindow
	if window <= 0 {
		window = DefaultValidWindow
	}

	if p.VerificationWindow == 0 {
		p.VerificationWindow = 24 * time.Hour
	}
	activeRecord, err := q.LookupActiveKycForUser(ctx, &p.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err := q.CreateKycRecord(ctx, db.CreateKycRecordParams{
				ID:                 p.ID,
				ValidWindow:        window.String(),
				UserID:             &p.UserID,
				VerificationWindow: p.VerificationWindow.String(),
			})
			if err != nil {
				return db.LookupActiveKycForUserRow{}, err
			}
			return q.LookupActiveKycForUser(ctx, &p.UserID)
		}
	}

	return activeRecord, nil
}

// GetByUser returns the KYC record for a user.
func GetByUser(ctx context.Context, q *db.Queries, userID string) (contract.LookupKycRecord, error) {
	if userID == "" {
		return contract.LookupKycRecord{}, ErrEmptyUserID
	}
	row, err := q.LookupKycRecordByUser(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contract.LookupKycRecord{}, ErrKycNotFound
		}
		return contract.LookupKycRecord{}, err
	}

	return mapGetRow(row)
}

// UpdateDocuments replaces document references and resets the record to pending.
func UpdateDocuments(ctx context.Context, q *db.Queries, p UpdateDocumentsParams) error {
	if p.UserID == "" {
		return ErrEmptyUserID
	}
	identity, err := json.Marshal(p.IdentityDoc)
	if err != nil {
		return fmt.Errorf("marshal identity doc: %w", err)
	}
	address, err := json.Marshal(p.ProofOfAddress)
	if err != nil {
		return fmt.Errorf("marshal proof of address: %w", err)
	}
	return q.UpdateKycDocuments(ctx, db.UpdateKycDocumentsParams{
		UserID:         &p.UserID,
		IdentityDoc:    identity,
		ProofOfAddress: address,
	})
}

// MarkVerifying moves a pending/rejected/expired record into verifying.
func MarkVerifying(ctx context.Context, q *db.Queries, userID string) error {
	if userID == "" {
		return ErrEmptyUserID
	}
	return q.MarkKycVerifying(ctx, &userID)
}

// Verify marks a KYC record as verified with the given risk profile.
func Verify(ctx context.Context, q *db.Queries, p VerifyParams) error {
	if p.UserID == "" || p.VerifiedBy == "" {
		return ErrEmptyUserID
	}
	profile, err := toDbRiskProfile(p.RiskProfile)
	if err != nil {
		return err
	}
	return q.VerifyKyc(ctx, db.VerifyKycParams{
		UserID:      &p.UserID,
		VerifiedBy:  &p.VerifiedBy,
		RiskProfile: profile,
	})
}

// Reject marks a KYC record as rejected with a reason.
func Reject(ctx context.Context, q *db.Queries, p RejectParams) error {
	if p.UserID == "" || p.VerifiedBy == "" {
		return ErrEmptyUserID
	}
	if p.RejectionReason == "" {
		return ErrEmptyRejectionReason
	}
	return q.RejectKyc(ctx, db.RejectKycParams{
		UserID:          &p.UserID,
		VerifiedBy:      &p.VerifiedBy,
		RejectionReason: &p.RejectionReason,
	})
}

// SetStatus updates only the KYC status field.
func SetStatus(ctx context.Context, q *db.Queries, p SetStatusParams) error {
	if p.UserID == "" {
		return ErrEmptyUserID
	}
	status, err := toDbStatus(p.Status)
	if err != nil {
		return err
	}
	return q.SetKycStatus(ctx, db.SetKycStatusParams{
		UserID: &p.UserID,
		Status: status,
	})
}

// List returns all KYC records.
func List(ctx context.Context, q *db.Queries) ([]contract.LookupKycRecord, error) {
	rows, err := q.ListKycs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]contract.LookupKycRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := mapListRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// ListByRiskProfile returns KYC records filtered by risk profile.
func ListByRiskProfile(ctx context.Context, q *db.Queries, profile contract.KYCRiskProfile) ([]contract.LookupKycRecord, error) {
	dbProfile, err := toDbRiskProfile(profile)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListKycsByRiskProfile(ctx, dbProfile)
	if err != nil {
		return nil, err
	}
	out := make([]contract.LookupKycRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := mapListByRiskRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// StatusClaimForUser returns the KYC status string for JWT claims.
// Users without a KYC record get ClaimStatusAbsent.
func StatusClaimForUser(ctx context.Context, q *db.Queries, userID string) (string, error) {
	if userID == "" {
		return ClaimStatusAbsent, nil
	}
	row, err := q.LookupKycRecordByUser(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ClaimStatusAbsent, nil
		}
		return "", err
	}
	return string(row.Status), nil
}

func toDbRiskProfile(p contract.KYCRiskProfile) (db.KycRiskProfile, error) {
	switch p {
	case contract.KYCRiskLow:
		return db.KycRiskProfileLow, nil
	case contract.KYCRiskMedium:
		return db.KycRiskProfileMedium, nil
	case contract.KYCRiskHigh:
		return db.KycRiskProfileHigh, nil
	default:
		return "", ErrInvalidRiskProfile
	}
}

func toDbStatus(s contract.KYCStatus) (db.KycStatus, error) {
	switch s {
	case contract.KYCPending:
		return db.KycStatusPending, nil
	case contract.KYCVerifying:
		return db.KycStatusVerifying, nil
	case contract.KYCVerified:
		return db.KycStatusVerified, nil
	case contract.KYCRejected:
		return db.KycStatusRejected, nil
	case contract.KYCExpired:
		return db.KycStatusExpired, nil
	default:
		return "", ErrInvalidStatus
	}
}

func mapGetRow(row db.LookupKycRecordByUserRow) (contract.LookupKycRecord, error) {
	return mapRecord(
		row.ID,
		*row.UserID,
		string(row.Status),
		row.ExpiresAt.Time,
		row.VerifyBefore.Time,
		row.Usable,
	)
}

func mapListRow(row db.ListKycsRow) (contract.LookupKycRecord, error) {
	return mapRecord(
		row.ID,
		*row.UserID,
		string(row.Status),
		row.ExpiresAt.Time,
		row.VerifyBefore.Time,
		row.Usable,
	)
}

func mapListByRiskRow(row db.ListKycsByRiskProfileRow) (contract.LookupKycRecord, error) {
	return mapRecord(
		row.ID, *row.UserID, string(row.Status), row.ExpiresAt.Time, row.VerifyBefore.Time, row.Usable,
	)
}

func mapRecord(
	id, userId, status string, expiresAt, verifyBefore time.Time, usable bool,
) (contract.LookupKycRecord, error) {
	rec := contract.LookupKycRecord{
		ID:           id,
		UserID:       userId,
		Status:       contract.KYCStatus(status),
		ExpiresAt:    expiresAt,
		VerifyBefore: verifyBefore,
		Usable:       usable,
	}
	return rec, nil
}

func formatTimestamptz(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}
