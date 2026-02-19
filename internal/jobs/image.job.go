package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"imagepp/internal/scheduler"

	"github.com/hibiken/asynq"
)

const (
	TypeImageProcess = "process:image"
)

type Job struct {
	ImageID    int64            `json:"image_id"`
	UserID     int64            `json:"user_id"`
	BucketName string           `json:"bucket_name"`
	ImageKey   string           `json:"image_key"`
	Operations []map[string]any `json:"operations"`
}

func EnqueueImageJob(ctx context.Context, client any, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	switch c := client.(type) {
	case *scheduler.Client:
		return c.Enqueue(ctx, TypeImageProcess, payload,
			asynq.MaxRetry(3),
			asynq.Timeout(10*time.Minute),
			asynq.Queue("critical"),
		)
	default:
		return fmt.Errorf("unsupported client type: %T", client)
	}
}
