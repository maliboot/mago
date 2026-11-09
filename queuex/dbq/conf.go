package dbq

import (
	"context"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/maliboot/mago/dispatcher"
	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex/logq"
)

type QueueErrHandler func(ctx context.Context, name string, t *dispatcher.TaskInfo, err error)
type QueueSuccessHandler func(ctx context.Context, name string, t *dispatcher.TaskInfo)

type Conf struct {
	CacheDir      string
	Concurrency   int
	MaxTry        int                      // 队列[最大重试]
	QueuesWeight  map[string]int           // 队列[权重]
	QueuesTimeout map[string]time.Duration // 队列[超时]

	errorHandler   QueueErrHandler
	successHandler QueueSuccessHandler
}

func NewConf(cacheDir string, concurrency int, queuesWeight map[string]int) *Conf {
	cacheDir = path.Join(cacheDir, "db")
	exist := eghelper.FileExists(cacheDir)
	if !exist {
		err := os.MkdirAll(cacheDir, 0750)
		if err != nil {
			logq.Log.Fatalf("QueueCacheDir[%s]Err: %v", cacheDir, err)
		}
	}
	if concurrency == 0 {
		concurrency = runtime.NumCPU() * 3 // 不超过4倍
	}
	if queuesWeight == nil {
		queuesWeight = map[string]int{"default": 1}
	}
	return &Conf{
		CacheDir:      cacheDir,
		Concurrency:   concurrency,
		QueuesWeight:  queuesWeight,
		MaxTry:        1,
		QueuesTimeout: make(map[string]time.Duration),
		errorHandler: func(ctx context.Context, name string, t *dispatcher.TaskInfo, err error) {
			//logq.Log.Errorf("dbQueue任务 %s 处理失败: body=[%+v],err=%v", name, t, err)
		},
	}
}

type dbMsg struct {
	ID       string `json:"id"`
	Queue    string `json:"queue"`
	Body     []byte `json:"body"` // payload 内容
	Received int    `json:"received"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

type Log struct {
}

func (l Log) Info(msg string, args ...any) {
	logq.Log.Infof(msg, args...)
}
