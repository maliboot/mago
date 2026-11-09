package dbq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maliboot/mago/dispatcher"
	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex/archiveq"
	"github.com/maliboot/mago/queuex/logq"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc

	queues             *Queues
	handlers           sync.Map
	aggregatorHandlers sync.Map
	aggregatorMaxSize  int
	aggregatorMaxDelay time.Duration

	archiver                      *archiveq.Archiver
	archiverSuccessMessagesCache  []*dispatcher.TaskInfo
	archiverSuccessMessagesLocker sync.Mutex
}

func NewServer(cacheDir string, concurrency int, aggregatorMaxSize int, aggregatorMaxDelay time.Duration) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	conf := NewConf(cacheDir, concurrency, nil)

	ins := &Server{
		ctx:                          ctx,
		cancel:                       cancel,
		queues:                       NewQueues(ctx, conf),
		archiver:                     archiveq.NewArchiver(ctx, cacheDir),
		archiverSuccessMessagesCache: make([]*dispatcher.TaskInfo, 0),
		aggregatorMaxSize:            aggregatorMaxSize,
		aggregatorMaxDelay:           aggregatorMaxDelay,
	}
	ins.initialize()
	return ins
}

func (s *Server) initialize() {
	eghelper.Cron(s.ctx, "DBQueue归档-探测器", time.Minute*5, func(ctx context.Context) {
		// 异常消息归档
		go s.flushMessages(false)

		// 成功消息归档
		s.flushMessages(true)
	})

	s.queues.c.successHandler = func(ctx context.Context, name string, t *dispatcher.TaskInfo) {
		if t.Retention == nil {
			return
		}
		s.archiverSuccessMessagesLocker.Lock()
		s.archiverSuccessMessagesCache = append(s.archiverSuccessMessagesCache, t)
		if len(s.archiverSuccessMessagesCache) < 100 {
			s.archiverSuccessMessagesLocker.Unlock()
			return
		}
		s.archiverSuccessMessagesLocker.Unlock()

		// flush
		s.flushMessages(true)
	}
}

func (s *Server) flushMessages(isSuccess bool) int {
	defer eghelper.PanicRecover("QueueFlushMessages")
	var msgList []*dispatcher.TaskInfo
	var err error
	if isSuccess {
		s.archiverSuccessMessagesLocker.Lock()
		msgList = make([]*dispatcher.TaskInfo, len(s.archiverSuccessMessagesCache))
		copy(msgList, s.archiverSuccessMessagesCache)
		s.archiverSuccessMessagesCache = s.archiverSuccessMessagesCache[:0]
		s.archiverSuccessMessagesLocker.Unlock()
	} else {
		msgList, err = s.queues.flushFailureMessages(100)
	}
	if err != nil {
		logq.Log.Errorf("flush[%v]MessagesErr.0:%v", isSuccess, err)
		return 0
	}
	msgListLen := len(msgList)
	if msgListLen == 0 {
		return 0
	}

	var archiveMsgList = make([]*archiveq.ArchiverMessage, msgListLen)
	for i, msg := range msgList {
		msgRet := 0
		createTime := time.Now()
		if msg.NextAt != nil {
			createTime = *msg.NextAt
		}

		if createTime.IsZero() {
			createTime = time.Now()
		}
		createTimeStr := createTime.Format(time.DateTime)
		if isSuccess {
			msgRet = 1
		}
		msgId := ""
		if msg.ID != nil {
			msgId = *msg.ID
		}
		queueName := ""
		if msg.Worker != nil {
			queueName = *msg.Worker
		}
		archiveMsgList[i] = &archiveq.ArchiverMessage{
			TaskID:    msgId,
			Name:      queueName,
			Body:      msg.Payload,
			TaskGroup: msg.GetGuid(),
			Result:    msgRet,
			ResultTxt: msg.StateMsg,
			Ext:       "",
			Created:   createTimeStr,
			Updated:   createTimeStr,
		}
	}
	err = s.archiver.Add(isSuccess, archiveMsgList)
	if err != nil {
		logq.Log.Errorf("flushQueueMessages[%v]Err.1:%v", isSuccess, err)
		return 0
	}

	return msgListLen
}

func (s *Server) RegisterHandler(taskName string, handler dispatcher.Handler) {
	s.RegisterHandleFunc(taskName, handler.ProcessTask)
}

func (s *Server) RegisterHandleFunc(taskName string, handler dispatcher.HandlerFunc) {
	s.handlers.Store(taskName, handler)
}

func (s *Server) RegisterQueues(m map[string]int) {
	for k, v := range m {
		s.queues.c.QueuesWeight[k] = v
	}
	queueLen := len(s.queues.c.QueuesWeight)
	if s.queues.c.Concurrency < queueLen {
		s.queues.c.Concurrency = queueLen
	}
}

func (s *Server) RegisterQueuesTimeout(m map[string]time.Duration) {
	for k, v := range m {
		s.queues.c.QueuesTimeout[k] = v
	}
}

func (s *Server) RegisterAggregator() {
	if s.aggregatorMaxSize == 0 || s.aggregatorMaxDelay == 0 {
		return
	}
	aggregatorHandlersMap := make(map[string]dispatcher.AggregatorHandlerFunc)
	s.aggregatorHandlers.Range(func(key, value any) bool {
		aggregatorHandlersMap[key.(string)] = value.(dispatcher.AggregatorHandlerFunc)
		return true
	})

	s.queues.initAggregator(s.aggregatorMaxSize, s.aggregatorMaxDelay, s.aggregatorMaxDelay, aggregatorHandlersMap)
}

func (s *Server) RegisterAggregatorHandler(group string, handler dispatcher.AggregatorHandlerFunc) {
	s.aggregatorHandlers.Store(group, handler)
}

func (s *Server) RegisterErrorHandlerFunc(handlerFunc dispatcher.ErrorHandlerFunc) {
	s.queues.c.errorHandler = func(ctx context.Context, name string, t *dispatcher.TaskInfo, err error) {
		handlerFunc(ctx, &t.Task, err)
	}
}

func (s *Server) ArchiverGet(ctx context.Context, isSuccess bool, page, pageSize int, ext map[string]string) ([]*archiveq.ArchiverMessage, error) {
	taskGroup := ""
	name := ""
	if len(ext) != 0 {
		taskGroup = ext["taskGroup"]
		name = ext["name"]
	}
	return s.archiver.Get(ctx, isSuccess, name, taskGroup, page, pageSize)
}

func (s *Server) ArchiverFlush(isSuccess bool) int {
	return s.flushMessages(isSuccess)
}

func (s *Server) Start(ctx context.Context) error {
	// 聚合处理
	s.RegisterAggregator()

	// 队列集启动
	handlers := make(map[string]dispatcher.HandlerFunc)
	s.handlers.Range(func(key, value any) bool {
		handlers[key.(string)] = value.(dispatcher.HandlerFunc)
		return true
	})
	err := s.queues.start(handlers)
	if err != nil {
		return fmt.Errorf("DBQueue初始化异常: %v", err)
	}

	// 队列归档
	err = s.archiver.Start()
	if err != nil {
		return fmt.Errorf("DBQueueArchiver初始化异常: %v", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
