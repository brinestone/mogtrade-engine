package eventpayloads

import "time"

type UserCreatedEventArgs struct {
	UserId    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}
