package repository

import (
	"context"
	"errors"

	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/dispatcher"
)

// Queue 是异步任务发布适配器：把 dispatcher.Task 投给 mago queuex（优先 redis 实现，README §12.3）。
// 队列服务由组合根在启动时初始化（conf/server/queue.go），这里只取客户端，不自己建连接池。
type Queue struct {
	conf *config.Conf
}

// 非单例：按配置装配，由组合根创建一次。
func NewQueue(conf *config.Conf) *Queue {
	return &Queue{conf: conf}
}

// Enqueue 投递任务；队列未启用或未初始化时返回明确错误，由调用方决定是否降级。
func (q *Queue) Enqueue(ctx context.Context, task *dispatcher.Task) error {
	if q.conf == nil || q.conf.Queue == nil {
		return errors.New("队列未配置")
	}

	client, err := q.conf.Queue.GetClient()
	if err != nil {
		return err
	}

	_, err = client.EnqueueContext(ctx, task)

	return err
}
