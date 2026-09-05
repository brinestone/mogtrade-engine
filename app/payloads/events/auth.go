package eventpayloads

import (
	"time"

	"github.com/brinestone/mogtrade/infra/db"
)

type UserSignedInEventArgs struct {
	// UserId    string             `json:"userId"`
	Timestamp  time.Time          `json:"timestamp"`
	Identifier string             `json:"identifier"`
	Provider   db.AccountProvider `json:"provider"`
}
type UserCreatedEventArgs struct {
	UserId    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}
