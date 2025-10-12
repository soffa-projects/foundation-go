package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

// AsynqQueue implements EmailQueue, PhoneQueue, and JobInspector using Asynq
type AsynqQueue struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// NewAsynqQueue creates a new Asynq queue client
func NewAsynqQueueProvider(redisURL string) (*AsynqQueue, error) {

	cfg, err := h.ParseUrl(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	opt := asynq.RedisClientOpt{
		Addr:     cfg.Host,
		Username: cfg.User,
		Password: cfg.Password,
		DB:       int(cfg.QueryWithDefault("db", 0).(int64)),
	}
	client := asynq.NewClient(opt)
	inspector := asynq.NewInspector(opt)
	return &AsynqQueue{
		client:    client,
		inspector: inspector,
	}, nil
}

func MustNewAsynqQueueProvider(redisURL string) *AsynqQueue {
	queue, err := NewAsynqQueueProvider(redisURL)
	if err != nil {
		panic(err)
	}
	return queue
}

// Feature-specific adapters have been moved to their respective feature packages
// to avoid import cycles

// Enqueue is the public implementation for enqueueing jobs (implements QueueClient interface)
func (q *AsynqQueue) Enqueue(ctx context.Context, jobType f.JobType, data any) (string, error) {
	// Serialize data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job data: %w", err)
	}

	// Create task
	task := asynq.NewTask(jobType, payload)

	// Enqueue task
	info, err := q.client.EnqueueContext(ctx, task,
		asynq.Queue("default"),
		asynq.MaxRetry(3),
	)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info("job enqueued id=%s (%s)", info.ID, jobType)

	return info.ID, nil
}

// EnqueueJob is an alias for Enqueue to match the QueueClient interface in provider.go
func (q *AsynqQueue) EnqueueJob(ctx context.Context, jobType f.JobType, data any) (string, error) {
	return q.Enqueue(ctx, jobType, data)
}

// GetJobStatus retrieves the status of a job by ID (implements JobInspector interface)
func (q *AsynqQueue) GetJobStatus(ctx context.Context, jobID string) (*f.JobStatus, error) {
	// Try to find the task in different queues and states
	// Asynq stores tasks in a "default" queue by default
	queues := []string{"default"}

	for _, queue := range queues {
		// Try to get task info
		taskInfo, err := q.inspector.GetTaskInfo(queue, jobID)
		if err != nil {
			// Task not found in this queue, continue
			continue
		}

		// Convert Asynq TaskInfo to our JobStatus
		status := &f.JobStatus{
			ID:        taskInfo.ID,
			Status:    mapTaskState(taskInfo.State),
			Type:      taskInfo.Type,
			Retried:   taskInfo.Retried,
			MaxRetry:  taskInfo.MaxRetry,
			LastError: taskInfo.LastErr,
		}

		// Set optional fields
		if !taskInfo.CompletedAt.IsZero() {
			status.CompletedAt = &taskInfo.CompletedAt
		}
		if !taskInfo.NextProcessAt.IsZero() {
			status.NextProcessAt = &taskInfo.NextProcessAt
		}

		log.Debug("job status retrieved id=%s status=%s", jobID, status.Status)

		return status, nil
	}

	// Task not found in any queue
	return nil, fmt.Errorf("job not found: %s", jobID)
}

func (q *AsynqQueue) Close() error {
	return q.client.Close()
}

// mapTaskState converts Asynq TaskState to our status string
func mapTaskState(state asynq.TaskState) string {
	switch state {
	case asynq.TaskStatePending:
		return "pending"
	case asynq.TaskStateActive:
		return "active"
	case asynq.TaskStateScheduled:
		return "scheduled"
	case asynq.TaskStateRetry:
		return "retry"
	case asynq.TaskStateArchived:
		return "archived"
	case asynq.TaskStateCompleted:
		return "completed"
	default:
		return "unknown"
	}
}
