package kyc

import (
	"context"
	"testing"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
)

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateRecord empty user ID", func(t *testing.T) {
		_, err := CreateRecord(ctx, nil, CreateParams{})
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID, got %v", err)
		}
	})

	t.Run("GetByUser empty user ID", func(t *testing.T) {
		_, err := GetByUser(ctx, nil, "")
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID, got %v", err)
		}
	})

	t.Run("GetByID empty ID", func(t *testing.T) {
		_, err := GetByID(ctx, nil, "")
		if err == nil {
			t.Fatal("expected error for empty ID, got nil")
		}
	})

	t.Run("UpdateProfile empty user ID", func(t *testing.T) {
		err := UpdateProfile(ctx, nil, UpdateProfileParams{})
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID, got %v", err)
		}
	})

	t.Run("UpdateProfile empty legal names", func(t *testing.T) {
		err := UpdateProfile(ctx, nil, UpdateProfileParams{
			UserID: "user_123",
		})
		if err == nil || err.Error() != "legal names is required" {
			t.Fatalf("expected 'legal names is required', got %v", err)
		}
	})

	t.Run("UpdateProfile invalid gender", func(t *testing.T) {
		err := UpdateProfile(ctx, nil, UpdateProfileParams{
			UserID:     "user_123",
			LegalNames: "John Doe",
			Gender:     "alien",
			Dob:        time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		})
		if err == nil || err.Error() != "invalid gender" {
			t.Fatalf("expected 'invalid gender', got %v", err)
		}
	})

	t.Run("UpdateDocuments empty user ID", func(t *testing.T) {
		err := UpdateDocuments(ctx, nil, UpdateDocumentsParams{})
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID, got %v", err)
		}
	})

	t.Run("Verify empty parameters", func(t *testing.T) {
		err := Verify(ctx, nil, VerifyParams{})
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID, got %v", err)
		}

		err = Verify(ctx, nil, VerifyParams{UserID: "u1"})
		if err != ErrEmptyUserID {
			t.Fatalf("expected ErrEmptyUserID when VerifiedBy is empty, got %v", err)
		}
	})

	t.Run("Reject empty rejection reason", func(t *testing.T) {
		err := Reject(ctx, nil, RejectParams{
			UserID:     "u1",
			VerifiedBy: "verifier1",
		})
		if err != ErrEmptyRejectionReason {
			t.Fatalf("expected ErrEmptyRejectionReason, got %v", err)
		}
	})

	t.Run("StatusClaimForUser empty user ID", func(t *testing.T) {
		claim, err := StatusClaimForUser(ctx, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim != ClaimStatusAbsent {
			t.Fatalf("expected claim %s, got %s", ClaimStatusAbsent, claim)
		}
	})
}

func TestStatusAndRiskProfileMappings(t *testing.T) {
	t.Run("toDbRiskProfile", func(t *testing.T) {
		p, err := toDbRiskProfile(contract.KYCRiskLow)
		if err != nil || string(p) != "low" {
			t.Errorf("failed mapping low risk: %v, %s", err, p)
		}
		p, err = toDbRiskProfile(contract.KYCRiskMedium)
		if err != nil || string(p) != "medium" {
			t.Errorf("failed mapping medium risk: %v, %s", err, p)
		}
		p, err = toDbRiskProfile(contract.KYCRiskHigh)
		if err != nil || string(p) != "high" {
			t.Errorf("failed mapping high risk: %v, %s", err, p)
		}
		_, err = toDbRiskProfile("invalid")
		if err != ErrInvalidRiskProfile {
			t.Errorf("expected ErrInvalidRiskProfile, got %v", err)
		}
	})

	t.Run("toDbStatus", func(t *testing.T) {
		s, err := toDbStatus(contract.KYCPending)
		if err != nil || string(s) != "pending" {
			t.Errorf("failed mapping pending status: %v, %s", err, s)
		}
		s, err = toDbStatus(contract.KYCVerifying)
		if err != nil || string(s) != "verifying" {
			t.Errorf("failed mapping verifying status: %v, %s", err, s)
		}
		s, err = toDbStatus(contract.KYCVerified)
		if err != nil || string(s) != "verified" {
			t.Errorf("failed mapping verified status: %v, %s", err, s)
		}
		s, err = toDbStatus(contract.KYCRejected)
		if err != nil || string(s) != "rejected" {
			t.Errorf("failed mapping rejected status: %v, %s", err, s)
		}
		s, err = toDbStatus(contract.KYCExpired)
		if err != nil || string(s) != "expired" {
			t.Errorf("failed mapping expired status: %v, %s", err, s)
		}
		_, err = toDbStatus("invalid")
		if err != ErrInvalidStatus {
			t.Errorf("expected ErrInvalidStatus, got %v", err)
		}
	})
}
