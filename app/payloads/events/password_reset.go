package eventpayloads

import "time"

// PasswordResetRequestedEventArgs is published when a user requests a password reset.
// The handler that publishes this event should include the user's identifier,
// email address, and the one‑time reset URL (which already contains the /api/v1 base path).
type PasswordResetRequestedEventArgs struct {
	// Timestamp indicates when the event was generated.
	Timestamp time.Time `json:"timestamp"`

	// UserID is the ULID of the user (stored as a string).
	UserID string `json:"userId"`

	// Email is the user's verified email address on file.
	Email string `json:"email"`

	// ResetURL is the absolute URL the user will click; it already includes the
	// project‑wide base path /api/v1/auth/password-reset/confirm?token=...
	ResetURL string `json:"resetUrl"`
}