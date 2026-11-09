package queuex

import (
	"context"

	"github.com/maliboot/mago/dispatcher"
)

type Client interface {
	Enqueue(task *dispatcher.Task) (*dispatcher.TaskInfo, error)
	EnqueueContext(ctx context.Context, task *dispatcher.Task) (*dispatcher.TaskInfo, error)
}
