package f

import (
	"context"
	"time"
)

type JobType = string

type JobStatus struct {
	ID            string
	Status        string
	Type          string
	Retried       int
	MaxRetry      int
	LastError     string
	CompletedAt   *time.Time
	NextProcessAt *time.Time
}

type QueueClient interface {
	Enqueue(ctx context.Context, jobType string, data any) (string, error)
	Close() error
}
