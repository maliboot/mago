package redisq

import (
	"github.com/maliboot/mago/dispatcher"

	"github.com/hibiken/asynq"
)

func ConvertToRedisTask(task *dispatcher.Task) *asynq.Task {
	if task == nil {
		return nil
	}
	cfg := dispatcher.GetTaskOpts(task)
	var newOpts = make([]asynq.Option, 0)
	if cfg.ID != nil {
		newOpts = append(newOpts, asynq.TaskID(*cfg.ID))
	}
	if cfg.MaxRetries != nil {
		newOpts = append(newOpts, asynq.MaxRetry(*cfg.MaxRetries))
	}
	if cfg.Timeout != nil {
		newOpts = append(newOpts, asynq.Timeout(*cfg.Timeout))
	}
	if cfg.NextAt != nil {
		newOpts = append(newOpts, asynq.ProcessAt(*cfg.NextAt))
	}
	if cfg.Retention != nil {
		newOpts = append(newOpts, asynq.Retention(*cfg.Retention))
	}
	if cfg.Worker != nil {
		newOpts = append(newOpts, asynq.Queue(*cfg.Worker))
	}
	if cfg.Aggregator != nil {
		newOpts = append(newOpts, asynq.Group(*cfg.Aggregator))
	}

	return asynq.NewTaskWithHeaders(task.Name, task.Payload, task.Headers, newOpts...)
}

func ConvertToDispatcherTaskList(taskList []*asynq.Task) []*dispatcher.Task {
	taskListLen := len(taskList)
	ret := make([]*dispatcher.Task, len(taskList))
	if taskListLen == 0 {
		return ret
	}
	for i, task := range taskList {
		ret[i] = dispatcher.NewTask(task.Type(), task.Headers(), task.Payload())
	}

	return ret
}
