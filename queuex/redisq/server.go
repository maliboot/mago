package redisq

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"

	"github.com/maliboot/mago/dispatcher"
	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex/archiveq"
	"github.com/maliboot/mago/queuex/logq"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc

	srv *asynq.Server
	mux *asynq.ServeMux

	config   *asynq.Config
	redisOpt asynq.RedisConnOpt

	handlers           sync.Map
	aggregatorHandlers sync.Map
	aggregatorMaxSize  int
	aggregatorMaxDelay time.Duration

	inspector *asynq.Inspector
	archiver  *archiveq.Archiver
}

func NewServer(redisOpt asynq.RedisConnOpt, cacheDir string, concurrency int, aggregatorMaxSize int, aggregatorMaxDelay time.Duration) *Server {
	if concurrency == 0 {
		concurrency = runtime.NumCPU() * 3 // 不超过4倍
	}
	config := asynq.Config{
		Concurrency: concurrency,
		Queues:      map[string]int{"default": 1},
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			return asynq.DefaultRetryDelayFunc(n, e, t)
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
			logq.Log.Errorf("redisQueue任务 %s 处理失败: body=[%s], err=%v", t.Type(), t.Payload(), err)
		}),
		Logger: logq.Log,
	}

	ctx, cancel := context.WithCancel(context.Background())
	ins := &Server{
		ctx:                ctx,
		cancel:             cancel,
		redisOpt:           redisOpt,
		config:             &config,
		inspector:          asynq.NewInspector(redisOpt),
		archiver:           archiveq.NewArchiver(ctx, cacheDir),
		aggregatorMaxSize:  aggregatorMaxSize,
		aggregatorMaxDelay: aggregatorMaxDelay,
	}
	ins.initialize()
	return ins
}

func (r *Server) initialize() {
	r.RegisterAggregator()
}

func (r *Server) initServer() {
	if r.srv != nil {
		return
	}

	r.srv = asynq.NewServer(r.redisOpt, *r.config)
	r.mux = asynq.NewServeMux()
	r.handlers.Range(func(router, handler any) bool {
		r.mux.HandleFunc(router.(string), func(ctx context.Context, task *asynq.Task) error {
			return handler.(dispatcher.HandlerFunc)(ctx, dispatcher.NewTask(task.Type(), task.Headers(), task.Payload()))
		})
		return true
	})
}

func (r *Server) Start(_ context.Context) error {
	eghelper.Cron(r.ctx, "DBQueue归档=探测器", time.Minute*5, func(ctx context.Context) {
		// 异常消息归档
		r.flushAllQueueMessages(false)
		// 成功消息归档
		r.flushAllQueueMessages(true)
	})

	time.Sleep(time.Second * 3) // 留点时间给其它 Servers 注册 handlers和queues
	r.initServer()
	err := r.srv.Ping()
	if err != nil {
		return fmt.Errorf("redis队列异常: %v", err)
	}

	err = r.archiver.Start()
	if err != nil {
		return fmt.Errorf("redis队列归档器异常: %v", err)
	}
	return r.srv.Start(r.mux)
}

func (r *Server) Stop(_ context.Context) error {
	if r.srv != nil {
		r.srv.Shutdown()
	}
	if r.archiver != nil {
		r.archiver.Stop()
	}
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *Server) flushAllQueueMessages(isSuccess bool) int {
	if isSuccess {
		return 0 // 暂时不存储redis的成功数据
	}

	retRows := 0
	batchSize := 300
	queueNames, err := r.inspector.Queues()
	if err != nil {
		logq.Log.Errorf("flushAllQueueMessagesErr[%v].1:%v", isSuccess, err)
		return retRows
	}
	if len(queueNames) == 0 {
		return retRows
	}

	for _, s := range queueNames {
		retRows += r.flushQueueMessages(s, batchSize, isSuccess)
	}

	return retRows
}

func (r *Server) flushQueueMessages(queue string, limit int, isSuccess bool) int {
	var msgList []*asynq.TaskInfo
	var err error
	if isSuccess {
		msgList, err = r.inspector.ListCompletedTasks(queue, asynq.PageSize(limit))
	} else {
		msgList, err = r.inspector.ListArchivedTasks(queue, asynq.PageSize(limit))
	}
	if err != nil {
		logq.Log.Errorf("flushQueueMessages[%v]MessagesErr.10:%v", isSuccess, err)
		return 0
	}
	msgListLen := len(msgList)
	if msgListLen == 0 {
		return 0
	}

	var archiveMsgList = make([]*archiveq.ArchiverMessage, msgListLen)
	for i, msg := range msgList {
		msgRet := 1
		createTime := msg.NextProcessAt
		if !isSuccess {
			msgRet = 0
			createTime = msg.LastFailedAt
		}
		if createTime.IsZero() {
			createTime = time.Now()
		}
		createTimeStr := createTime.Format(time.DateTime)
		archiveMsgList[i] = &archiveq.ArchiverMessage{
			TaskID:    msg.ID,
			Name:      msg.Queue,
			Body:      msg.Payload,
			TaskGroup: msg.Headers[dispatcher.TaskHeadGuid],
			Result:    msgRet,
			ResultTxt: msg.LastErr,
			Ext:       "",
			Created:   createTimeStr,
			Updated:   createTimeStr,
		}
	}
	err = r.archiver.Add(isSuccess, archiveMsgList)
	if err != nil {
		logq.Log.Errorf("flushQueueMessages[%v]Err.11:%v", isSuccess, err)
		return 0
	}

	if !isSuccess {
		for _, msg := range msgList {
			err = r.inspector.DeleteTask(queue, msg.ID)
			if err != nil {
				logq.Log.Errorf("flushQueueMessages[%v]Err.12:%v", isSuccess, err)
			}
		}
	}

	return msgListLen
}

func (r *Server) RegisterHandleFunc(pattern string, handler dispatcher.HandlerFunc) {
	r.handlers.Store(pattern, handler)
}

func (r *Server) RegisterHandler(pattern string, handler dispatcher.Handler) {
	r.RegisterHandleFunc(pattern, handler.ProcessTask)
}

func (r *Server) RegisterQueues(queues map[string]int) {
	for k, v := range queues {
		r.config.Queues[k] = v
	}
	queueLen := len(r.config.Queues)
	if r.config.Concurrency < queueLen {
		r.config.Concurrency = queueLen
	}
}

func (r *Server) RegisterQueuesTimeout(map[string]time.Duration) {
	return
}

func (r *Server) RegisterAggregatorHandler(group string, handler dispatcher.AggregatorHandlerFunc) {
	r.aggregatorHandlers.Store(group, handler)
}

func (r *Server) RegisterAggregator() {
	if r.aggregatorMaxSize == 0 || r.aggregatorMaxDelay == 0 {
		return
	}
	r.config.GroupAggregator = asynq.GroupAggregatorFunc(func(group string, tasks []*asynq.Task) *asynq.Task {
		handler, ok := r.aggregatorHandlers.Load(group)
		if !ok {
			// 多级匹配
			if !strings.Contains(group, ":") {
				return nil
			}
			groupSlices := strings.Split(group, ":")
			handler2, ok2 := r.aggregatorHandlers.Load(groupSlices[0])
			if !ok2 {
				return nil
			}
			handler = handler2
		}
		dispatcherTask := handler.(dispatcher.AggregatorHandlerFunc)(group, ConvertToDispatcherTaskList(tasks))
		return ConvertToRedisTask(dispatcherTask)
	})
	r.config.GroupMaxSize = r.aggregatorMaxSize
	r.config.GroupMaxDelay = r.aggregatorMaxDelay

	aggregatorMaxDelay := r.aggregatorMaxDelay / 2
	if aggregatorMaxDelay < time.Second {
		aggregatorMaxDelay = time.Second
	}
	r.config.GroupGracePeriod = aggregatorMaxDelay
}

func (r *Server) RegisterErrorHandlerFunc(handlerFunc dispatcher.ErrorHandlerFunc) {
	r.config.ErrorHandler = asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
		handlerFunc(ctx, ConvertToDispatcherTaskList([]*asynq.Task{task})[0], err)
	})
}

func (r *Server) ArchiverGet(ctx context.Context, isSuccess bool, page, pageSize int, ext map[string]string) ([]*archiveq.ArchiverMessage, error) {
	taskGroup := ""
	name := ""
	if len(ext) != 0 {
		taskGroup = ext["taskGroup"]
		name = ext["name"]
	}
	return r.archiver.Get(ctx, isSuccess, name, taskGroup, page, pageSize)
}

func (r *Server) ArchiverFlush(isSuccess bool) int {
	return r.flushAllQueueMessages(isSuccess)
}
