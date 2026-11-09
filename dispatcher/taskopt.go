package dispatcher

import "time"

type TaskOptions struct {
	ID         *string
	MaxRetries *int
	Timeout    *time.Duration
	NextAt     *time.Time
	Retention  *time.Duration
	Worker     *string
	Aggregator *string
}

type TaskOption func(*TaskOptions)

func TaskID(id string) TaskOption {
	return func(o *TaskOptions) {
		o.ID = &id
	}
}
func TaskMaxRetries(maxRetries int) TaskOption {
	return func(o *TaskOptions) {
		o.MaxRetries = &maxRetries
	}
}

func TaskTimeout(timeout time.Duration) TaskOption {
	return func(o *TaskOptions) {
		o.Timeout = &timeout
	}
}

func TaskNextAt(nextAt time.Time) TaskOption {
	return func(o *TaskOptions) {
		o.NextAt = &nextAt
	}
}

func TaskRetention(retention time.Duration) TaskOption {
	return func(o *TaskOptions) {
		o.Retention = &retention
	}
}

func TaskWorker(worker string) TaskOption {
	return func(o *TaskOptions) {
		o.Worker = &worker
	}
}

func TaskAggregator(aggregator string) TaskOption {
	return func(o *TaskOptions) {
		o.Aggregator = &aggregator
	}
}
