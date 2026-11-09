package queuex

import (
	"context"
	"time"

	"github.com/maliboot/mago/dispatcher"
	"github.com/maliboot/mago/queuex/archiveq"
)

type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	RegisterHandler(taskName string, handler dispatcher.Handler)
	RegisterHandleFunc(taskName string, handler dispatcher.HandlerFunc)
	RegisterQueues(map[string]int)                  // 设置队列，映射[队列名称]权重
	RegisterQueuesTimeout(map[string]time.Duration) // 只对db队列起作用
	RegisterAggregatorHandler(group string, handler dispatcher.AggregatorHandlerFunc)
	RegisterErrorHandlerFunc(handlerFunc dispatcher.ErrorHandlerFunc)

	ArchiverGet(ctx context.Context, isSuccess bool, page, pageSize int, ext map[string]string) ([]*archiveq.ArchiverMessage, error)
	ArchiverFlush(isSuccess bool) int
}
