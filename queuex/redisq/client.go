package redisq

import (
	"context"
	"time"

	"github.com/hibiken/asynq"

	"github.com/maliboot/mago/dispatcher"
)

type Client struct {
	clt              *asynq.Client
	aggregatorEnable bool
	maxRetention     time.Duration
}

func NewClient(redisOpt asynq.RedisConnOpt, aggregatorEnable bool) *Client {
	client := asynq.NewClient(redisOpt)

	return &Client{clt: client, aggregatorEnable: aggregatorEnable, maxRetention: time.Hour * 24 * 30}
}

func (c *Client) Enqueue(task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	return c.EnqueueContext(context.Background(), task)
}

func (c *Client) EnqueueContext(ctx context.Context, task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	taskOpt := dispatcher.GetTaskOpts(task)
	if !c.aggregatorEnable {
		taskOpt.Aggregator = nil
	}
	if taskOpt.Retention != nil && *taskOpt.Retention > c.maxRetention {
		tmpMaxRetention := c.maxRetention
		taskOpt.Retention = &tmpMaxRetention
	}
	taskInfo, err := c.clt.EnqueueContext(ctx, ConvertToRedisTask(task))
	if err != nil {
		return nil, err
	}
	var state dispatcher.TaskState
	if taskInfo.State == asynq.TaskStateCompleted {
		state = dispatcher.TaskStateSuccessful
	}
	switch taskInfo.State {
	case asynq.TaskStateCompleted:
		state = dispatcher.TaskStateSuccessful
		break
	case asynq.TaskStateArchived:
		state = dispatcher.TaskStateFailure
		break
	case asynq.TaskStateActive:
		state = dispatcher.TaskStateDoing
		break
	default:
		state = dispatcher.TaskStateReady
	}
	return &dispatcher.TaskInfo{
		Task: dispatcher.Task{
			Name:    taskInfo.Type,
			Payload: taskInfo.Payload,
			Headers: taskInfo.Headers,
		},
		TaskOptions: dispatcher.TaskOptions{
			ID:         &taskInfo.ID,
			MaxRetries: &taskInfo.MaxRetry,
			Timeout:    &taskInfo.Timeout,
			NextAt:     &taskInfo.NextProcessAt,
			Retention:  &taskInfo.Retention,
			Worker:     &taskInfo.Queue,
		},
		Retries:  taskInfo.Retried,
		State:    state,
		StateMsg: taskInfo.LastErr,
	}, nil
}
