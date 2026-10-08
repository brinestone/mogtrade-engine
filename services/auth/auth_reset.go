package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// ResetPasswordRequestInput holds the data needed to request a password reset.
type ResetPasswordRequestInput struct {
	Email         string
	ExpireMinutes int
}

// ResetPasswordRequestOutput is returned by the service so the controller can
// build the confirmation URL and publish the event.
// It contains the raw token that the service generated, and the user's email.
type ResetPasswordRequestOutput struct {
	RawToken  string // the one‑time token (base64url, 44 chars)
	UserEmail string // user's email – the controller will use it for the event
}

// ResetPasswordRequest creates a one‑time verification row and returns
// the raw token and user email.  It does NOT publish any event – that is
// the caller's responsibility (application layer).
func ResetPasswordRequest(ctx context.Context,
	q *db.Queries,
	input ResetPasswordRequestInput) (ResetPasswordRequestOutput, error) {

	// ---- 1️⃣ Resolve expiry (the only place we use the parameter) ----
	expiresAt := time.Now().Add(time.Duration(input.ExpireMinutes) * time.Minute)

	// ---- 2️⃣ Find the user --------------------------------------------
	user, err := q.FindUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResetPasswordRequestOutput{}, ErrUserNotFound
		}
		return ResetPasswordRequestOutput{}, err
	}

	// ---- 3️⃣ Generate a cryptographically‑secure random token ---------
	// 32 random bytes → base64url string (44 characters, no padding)
	rawTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rawTokenBytes); err != nil {
		return ResetPasswordRequestOutput{}, err
	}
	rawToken := base64.RawStdEncoding.EncodeToString(rawTokenBytes)

	// ---- 4️⃣ Hash the token with Argon2id ---------------------------
	// Re‑use the existing HashPassword helper – it generates a new salt and
	// returns a hex‑encoded "salt+hash" that we store in the verification row.
	hashedToken, err := HashPassword(rawToken)
	if err != nil {
		return ResetPasswordRequestOutput{}, err
	}

	// ---- 5️⃣ Store the verification row -----------------------------
	verParams := db.InsertVerificationParams{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      "email_reset",
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}
	if err := q.InsertVerification(ctx, verParams); err != nil {
		return ResetPasswordRequestOutput{}, err
	}

	// ---- 5️⃣ Return the raw token and user email (no event publishing) -
	return ResetPasswordRequestOutput{
		RawToken:  rawToken,
		UserEmail: user.Email,
	}, nil
}
