package dispatcher

import "context"

type TaskHandlers map[string]func(task *Task) error

type Handler interface {
	ProcessTask(context.Context, *Task) error
}

type HandlerFunc func(context.Context, *Task) error

func (fn HandlerFunc) ProcessTask(ctx context.Context, task *Task) error {
	return fn(ctx, task)
}

type AggregatorHandler interface {
	Aggregate(group string, tasks []*Task) *Task
}
type AggregatorHandlerFunc func(group string, tasks []*Task) *Task

func (fn AggregatorHandlerFunc) ProcessTask(group string, tasks []*Task) *Task {
	return fn(group, tasks)
}

type ErrorHandler interface {
	HandleError(ctx context.Context, task *Task, err error)
}
type ErrorHandlerFunc func(ctx context.Context, task *Task, err error)

func (fn ErrorHandlerFunc) HandleError(ctx context.Context, task *Task, err error) {
	fn(ctx, task, err)
}
