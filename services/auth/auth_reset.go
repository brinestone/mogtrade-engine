package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
)

// ResetPasswordRequestInput holds the data needed to request a password reset.
type ResetPasswordRequestInput struct {
	IpAddress      string
	CallbackUrl    string
	Email          string
	ValidWindow    time.Duration
	VerificationId string
}

// ResetPasswordRequestOutput is returned by the service so the controller can
// build the confirmation URL and publish the event.
// It contains the raw token that the service generated, and the user's email.
type ResetPasswordRequestOutput struct {
	RawToken  string // the one‑time token (base64url, 44 chars)
	UserEmail string // user's email – the controller will use it for the event
}

// BeginPasswordReset creates a one‑time verification row and returns
// the raw token and user email.  It does NOT publish any event – that is
// the caller's responsibility (application layer).
func BeginPasswordReset(ctx context.Context,
	q *db.Queries,
	input ResetPasswordRequestInput) (ResetPasswordRequestOutput, error) {

	user, err := q.FindUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResetPasswordRequestOutput{}, ErrUserNotFound
		}
		return ResetPasswordRequestOutput{}, err
	}

	// 32 random bytes → base64url string (44 characters, no padding)
	rawTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rawTokenBytes); err != nil {
		return ResetPasswordRequestOutput{}, err
	}
	rawToken := base64.RawStdEncoding.EncodeToString(rawTokenBytes)

	// Re‑use the existing HashPassword helper – it generates a new salt and
	// returns a hex‑encoded "salt+hash" that we store in the verification row.
	hashedToken, err := HashPassword(rawToken)
	if err != nil {
		return ResetPasswordRequestOutput{}, err
	}

	verParams := db.CreateVerificationParams{
		ID:          input.VerificationId,
		UserID:      user.ID,
		Token:       hashedToken,
		ValidWindow: input.ValidWindow.String(),
		Type:        db.VerificationTypeEmailReset,
		Ip:          input.IpAddress,
		CallbackUrl: &input.CallbackUrl,
	}
	if err := q.CreateVerification(ctx, verParams); err != nil {
		return ResetPasswordRequestOutput{}, err
	}

	return ResetPasswordRequestOutput{
		RawToken:  rawToken,
		UserEmail: user.Email,
	}, nil
}
