package dbq

import (
	"context"

	"github.com/maliboot/mago/dispatcher"
)

type Client struct {
	queues           *Queues
	aggregatorEnable bool
}

func NewClient(cacheDir string, aggregatorEnable bool) *Client {
	conf := NewConf(cacheDir, 0, nil)
	return &Client{
		queues:           NewQueues(context.Background(), conf),
		aggregatorEnable: aggregatorEnable,
	}
}

func (c *Client) EnqueueContext(ctx context.Context, task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	if !c.aggregatorEnable {
		dispatcher.GetTaskOpts(task).Aggregator = nil
	}
	return c.queues.pushWithContext(ctx, task)
}

func (c *Client) Enqueue(task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	return c.EnqueueContext(context.Background(), task)
}
