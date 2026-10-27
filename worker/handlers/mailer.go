package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brinestone/mogtrade/tasks"
	"github.com/hibiken/asynq"
)

func HandleEmailDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	// TODO: send mail
	return nil
}
