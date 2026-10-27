package tasks

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeEmailDelivery = "email:deliver"
)

type EmailDeliveryPayload struct {
	From             string `json:"from"`
	Subject          string `json:"subject"`
	To               string `json:"to"`
	Content          string `json:"content"`
	IdempotencyToken string `json:"idempotencyToken"`
}

func NewEmailDeliveryTask(p EmailDeliveryPayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeEmailDelivery, payload, asynq.MaxRetry(10), asynq.Timeout(30*time.Minute)), nil
}
