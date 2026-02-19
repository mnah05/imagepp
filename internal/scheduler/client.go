package scheduler

import (
	"context"

	"github.com/hibiken/asynq"
)

// Client wraps asynq.Client for task scheduling
type Client struct {
	client *asynq.Client
}

// NewClient creates a new scheduler client
func NewClient(redisOpt asynq.RedisClientOpt) *Client {
	return &Client{
		client: asynq.NewClient(redisOpt),
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue enqueues a task
func (c *Client) Enqueue(ctx context.Context, taskType string, payload []byte, opts ...asynq.Option) error {
	task := asynq.NewTask(taskType, payload)
	_, err := c.client.EnqueueContext(ctx, task, opts...)
	return err
}
