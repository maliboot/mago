package dbq

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	_ "embed"
	"encoding/gob"
	"fmt"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"

	"github.com/maliboot/mago/dispatcher"
	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex/logq"
)

//go:embed sqlite.sql
var schemeSql string

var (
	dbQueues *Queues
)

type QueueShard struct {
	ctx    context.Context
	cancel context.CancelFunc

	name   string
	no     int
	db     *sql.DB
	ins    *goqite.Queue
	runner *jobs.Runner

	maxTry         int
	errorHandler   QueueErrHandler
	successHandler QueueSuccessHandler
}

func NewQueueShard(ctx context.Context, name string, no int, cacheDir string, timeout time.Duration, maxTry int) (*QueueShard, error) {
	if maxTry == 0 {
		maxTry = 3
	}
	queueName := fmt.Sprintf("%s_%d", name, no)
	dbName := path.Join(cacheDir, fmt.Sprintf("%s_%d.db?_journal=WAL&_temp_store=MEMORY&_cache_size=-16000&_busy_timeout=5000", name, no))
	db, err := eghelper.SqliteOpen(dbName)
	if err != nil {
		return nil, fmt.Errorf("dbQueueShardInitErr.0: %v", err)
	}

	exists, err := eghelper.IsSqliteTableExists(db, "goqite")
	if err != nil {
		return nil, fmt.Errorf("dbQueueShardInitErr.1: %v", err)
	}
	if !exists {
		if _, err = db.Exec(schemeSql); err != nil {
			return nil, fmt.Errorf("dbQueueShardInitErr.2: %v", err)
		}
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	q := goqite.New(goqite.NewOpts{
		DB:         db,
		Name:       queueName,
		MaxReceive: maxTry, // 最多重试
		Timeout:    timeout,
		SQLFlavor:  goqite.SQLFlavorSQLite,
	})
	r := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        100,
		Log:          Log{},
		PollInterval: 100 * time.Millisecond,
		Queue:        q,
		Extend:       time.Second * 45, // 超时时间续期 timeout-extend
	})

	ctx, cancel := context.WithCancel(ctx)
	return &QueueShard{
		ctx:    ctx,
		cancel: cancel,
		name:   name,
		no:     no,
		db:     db,
		ins:    q,
		runner: r,
		maxTry: maxTry,
	}, nil
}

func (qs *QueueShard) push(ctx context.Context, task *dispatcher.Task) (string, error) {
	taskOpts := dispatcher.GetTaskOpts(task)
	delay := time.Duration(0)
	if taskOpts.NextAt != nil {
		nextAtInterval := taskOpts.NextAt.Sub(time.Now())
		if nextAtInterval > 0 {
			delay = nextAtInterval
		}
	}

	if taskOpts.ID == nil {
		taskCtx, err := task.Marshal()
		if err != nil {
			return "", fmt.Errorf("dbQueueTaskMarshalErr:%v", err)
		}
		id := fmt.Sprintf("m_%x", md5.Sum(taskCtx))
		taskOpts.ID = &id
	}

	body, err := qs.taskMarshal(dispatcher.NewTaskInfo(*task))
	if err != nil {
		return "", fmt.Errorf("dbQueueTaskInfoMarshalErr:%v", err)
	}
	jobId, err := jobs.Create(ctx, qs.ins, task.Name, goqite.Message{
		ID:       goqite.ID(*taskOpts.ID),
		Body:     body,
		Delay:    delay,
		Priority: 0,
	})
	if err != nil {
		return "", err
	}

	logq.Log.Infof("dbQueueShardPushMsgSuccess_%s=[%s]", *taskOpts.ID, jobId)
	return string(jobId), err
}

func (qs *QueueShard) taskMarshal(task *dispatcher.TaskInfo) ([]byte, error) {
	return task.Marshal()
}

func (qs *QueueShard) taskUnmarshal(raw []byte) (*dispatcher.TaskInfo, error) {
	return dispatcher.NewEmptyTaskInfo().Unmarshal(raw)
}

func (qs *QueueShard) handlerRegister(name string, handlerFunc func(ctx context.Context, t *dispatcher.TaskInfo) error) {
	qs.runner.Register(name, func(ctx context.Context, m []byte) error {
		task, err := qs.taskUnmarshal(m)
		if err != nil {
			return fmt.Errorf("dbQueueTaskUnmarshalErr: %v", err)
		}

		timeout := time.Second * 5
		if task.Timeout != nil {
			timeout = *task.Timeout
		}

		err = eghelper.DoTimeout(ctx, timeout, func(ctx context.Context) error { return handlerFunc(ctx, task) })

		// errHandler
		if err != nil && qs.errorHandler != nil {
			go qs.errorHandler(ctx, name, task, err)
			return err
		}

		// successHandler
		if err == nil && qs.successHandler != nil {
			go qs.successHandler(ctx, name, task)
			return nil
		}

		return err
	})
}

func (qs *QueueShard) setErrorHandler(fn QueueErrHandler) {
	qs.errorHandler = fn
}

func (qs *QueueShard) setSuccessHandler(fn QueueSuccessHandler) {
	qs.successHandler = fn
}

func (qs *QueueShard) flushFailureMessages(limit int) ([]*dispatcher.TaskInfo, error) {
	querySql := `
WITH failed AS (
    SELECT id, queue, body, received, created, updated
    FROM goqite
    WHERE received >= ?
    LIMIT ?
)
DELETE FROM goqite
WHERE id IN (SELECT id FROM failed)
RETURNING id, queue, body, received, created, updated;
`
	rows, err := qs.db.QueryContext(context.Background(), querySql, qs.maxTry, limit)
	if err != nil {
		return nil, fmt.Errorf("getAndDeletedJobsSqlErr: %v", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var msgs = make([]*dispatcher.TaskInfo, 0)
	for rows.Next() {
		var m dbMsg
		var t = dispatcher.NewEmptyTaskInfo()

		if err := rows.Scan(&m.ID, &m.Queue, &m.Body, &m.Received, &m.Created, &m.Updated); err != nil {
			return nil, fmt.Errorf("getAndDeletedJobsSqlScanErr: %v", err)
		}
		if len(m.Body) != 0 {
			var jm struct {
				Name    string
				Message []byte
			}
			err = gob.NewDecoder(bytes.NewReader(m.Body)).Decode(&jm)
			if err != nil {
				logq.Log.Errorf("getAndDeletedJobsSqlDecodeErr: %v", err)
			} else {
				m.Body = jm.Message
				taskUnmarshal, err := qs.taskUnmarshal(m.Body)
				if err != nil {
					logq.Log.Errorf("getAndDeletedJobsSqlUnmarshalErr: %v", err)
				} else {
					t = taskUnmarshal
				}
			}
		}

		t.Retries = m.Received
		t.ID = &m.ID
		t.State = dispatcher.TaskStateFailure
		msgs = append(msgs, t)
	}
	return msgs, nil
}

func (qs *QueueShard) start() {
	logq.Log.Debugf("数据库队列[%s]分片[%d]初始化工作完成", qs.name, qs.no)
	qs.runner.Start(qs.ctx)
}

func (qs *QueueShard) Stop() {
	qs.cancel()
	_ = qs.db.Close()
}

type Queue struct {
	name       string
	shards     []*QueueShard
	shardCount int

	rr atomic.Uint32
}

func NewQueue(
	ctx context.Context,
	name string,
	shardCount int,
	cacheDir string,
	timeout time.Duration,
	maxTry int,
	errorHandler QueueErrHandler,
	successHandler QueueSuccessHandler,
) (*Queue, error) {
	ins := &Queue{
		name:       name,
		shards:     make([]*QueueShard, shardCount),
		shardCount: shardCount,
	}

	var err error
	for i := 0; i < shardCount; i++ {
		ins.shards[i], err = NewQueueShard(ctx, name, i, cacheDir, timeout, maxTry)
		if err != nil {
			return nil, err
		}
		ins.shards[i].setSuccessHandler(successHandler)
		ins.shards[i].setErrorHandler(errorHandler)
	}

	return ins, nil
}

func (q *Queue) handlerRegister(name string, handlerFunc func(ctx context.Context, t *dispatcher.TaskInfo) error) {
	for _, shard := range q.shards {
		shard.handlerRegister(name, handlerFunc)
	}
}

func (q *Queue) push(ctx context.Context, task *dispatcher.Task) (string, error) {
	if q.shardCount == 0 {
		return "", fmt.Errorf("DBQueue分片还未初始化完成.1")
	}
	idx := q.rr.Add(1) % uint32(q.shardCount)
	return q.shards[idx].push(ctx, task)
}

func (q *Queue) flushFailureMessages(limit int) ([]*dispatcher.TaskInfo, error) {
	if q.shardCount == 0 {
		return nil, fmt.Errorf("DBQueue分片还未初始化完成.2")
	}
	var ret = make([]*dispatcher.TaskInfo, 0)
	for _, shard := range q.shards {
		shardMsgs, err := shard.flushFailureMessages(limit)
		if err != nil {
			return nil, err
		}
		ret = append(ret, shardMsgs...)
	}

	return ret, nil
}

func (q *Queue) start() {
	for _, shard := range q.shards {
		go shard.start()
	}
}

func (q *Queue) stop() {
	for _, shard := range q.shards {
		go shard.Stop()
	}
}

type Queues struct {
	ctx        context.Context
	c          *Conf
	m          sync.Map
	aggregator *dispatcher.Aggregator[dispatcher.Task]
}

func (qq *Queues) initQueues(handlers map[string]dispatcher.HandlerFunc) error {
	weightMap := qq.countWeight(qq.c.QueuesWeight, qq.c.Concurrency)
	for qn, count := range weightMap {
		q, err := NewQueue(qq.ctx, qn, count, qq.c.CacheDir, qq.c.QueuesTimeout[qn], qq.c.MaxTry, qq.c.errorHandler, qq.c.successHandler)
		if err != nil {
			return err
		}

		// 注册handler
		if len(handlers) != 0 {
			for handlerName, handler := range handlers {
				q.handlerRegister(handlerName, func(ctx context.Context, t *dispatcher.TaskInfo) error {
					return handler.ProcessTask(ctx, &t.Task)
				})
			}
		}

		qq.m.Store(qn, q)
	}
	return nil
}

func (qq *Queues) countWeight(weights map[string]int, total int) map[string]int {
	// 计算总权重
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	ret := make(map[string]int, len(weights))
	remaining := total

	// 第一轮：按比例分配，使用四舍五入，更公平
	for qn, w := range weights {
		count := (w*total*2 + totalWeight) / (totalWeight * 2)

		if count < 1 {
			count = 1
		}

		// 防止分配过多
		if count > remaining {
			count = remaining
		}

		ret[qn] = count
		remaining -= count
	}

	// 如果还有剩余并发，按权重从大到小依次补1
	if remaining > 0 {
		// 按权重降序补剩余
		type pair struct {
			name string
			w    int
		}
		ps := make([]pair, 0, len(weights))
		for qn, w := range weights {
			ps = append(ps, pair{qn, w})
		}

		// 简单排序：权重大的优先补
		for i := 0; i < len(ps)-1; i++ {
			for j := i + 1; j < len(ps); j++ {
				if ps[i].w < ps[j].w {
					ps[i], ps[j] = ps[j], ps[i]
				}
			}
		}

		for i := 0; i < remaining; i++ {
			ret[ps[i%len(ps)].name]++ // 轮询补给，避免都给同一个
		}
	}

	return ret
}

func (qq *Queues) initAggregator(maxSize int, maxDelay, gracePeriod time.Duration, aggregatorHandlers map[string]dispatcher.AggregatorHandlerFunc) {
	if qq.aggregator != nil {
		logq.Log.Errorf("aggregator already registered")
		return
	}

	flushTime := maxDelay
	if flushTime > gracePeriod {
		flushTime = gracePeriod
	}
	qq.aggregator = dispatcher.NewAggregator[dispatcher.Task](
		qq.ctx, "DB队列", flushTime,
		func(group string, tasks []*dispatcher.AggregatorEvent[dispatcher.Task]) {
			handler := aggregatorHandlers[group]
			if handler == nil {
				// 多级匹配
				if !strings.Contains(group, ":") {
					logq.Log.Errorf("aggregatorHandler.0 not found for %s", group)
					return
				}
				groupSlices := strings.Split(group, ":")
				handler2, ok2 := aggregatorHandlers[groupSlices[0]]
				if !ok2 {
					logq.Log.Errorf("aggregatorHandler.1 not found for %s", groupSlices[0])
					return
				}
				handler = handler2
			}

			dispatcherTasks := make([]*dispatcher.Task, len(tasks))
			for i, task := range tasks {
				dispatcherTasks[i] = &task.Data
			}
			aggregatorTaskRaw := handler(group, dispatcherTasks)
			if aggregatorTaskRaw == nil {
				logq.Log.Errorf("aggregatorHandler.2:聚合响应task为空")
				return
			}
			aggregatorTask := tasks[0].Data
			aggregatorTaskOpts := dispatcher.GetTaskOpts(&aggregatorTask)
			aggregatorTask.Name = aggregatorTaskRaw.Name
			aggregatorTask.Headers = aggregatorTaskRaw.Headers
			aggregatorTask.Payload = aggregatorTaskRaw.Payload
			aggregatorTaskOpts.Aggregator = nil
			_, err := qq.push(&aggregatorTask)
			if err != nil {
				logq.Log.Errorf("aggregatorHandler.3:%v", err)
			}
		})
	qq.aggregator.SetFlushNum(maxSize)
}

func (qq *Queues) pushWithContext(ctx context.Context, task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	taskOpts := dispatcher.GetTaskOpts(task)
	if taskOpts.Worker == nil {
		tmp := "default"
		taskOpts.Worker = &tmp
	}
	if taskOpts.NextAt == nil {
		nowTime := time.Now()
		taskOpts.NextAt = &nowTime
	}

	if taskOpts.Aggregator != nil && qq.aggregator != nil {
		qq.aggregator.Add(&dispatcher.AggregatorEvent[dispatcher.Task]{Key: *taskOpts.Aggregator, Data: *task})
		return &dispatcher.TaskInfo{
			Task: *task,
		}, nil
	}

	q, ok := qq.m.Load(*taskOpts.Worker)
	if !ok {
		return nil, fmt.Errorf("dbQueueEnqueueContextErr.0=queue %s not found", task.Name)
	}
	id, err := q.(*Queue).push(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("dbQueueEnqueueContextErr.2:%v", err)
	}
	return &dispatcher.TaskInfo{
		Task: *task,
		TaskOptions: dispatcher.TaskOptions{
			ID: &id,
		},
	}, nil
}

func (qq *Queues) push(task *dispatcher.Task) (*dispatcher.TaskInfo, error) {
	return qq.pushWithContext(context.Background(), task)
}

func (qq *Queues) flushFailureMessages(limit int) ([]*dispatcher.TaskInfo, error) {
	var ret = make([]*dispatcher.TaskInfo, 0)
	var err error
	qq.m.Range(func(key, value any) bool {
		queueMsgs, err := value.(*Queue).flushFailureMessages(limit)
		if err != nil {
			return false
		}
		ret = append(ret, queueMsgs...)
		return true
	})
	return ret, err
}

func (qq *Queues) Range(fn func(queueName string, queue *Queue) bool) {
	qq.m.Range(func(key, value any) bool {
		return fn(key.(string), value.(*Queue))
	})
}

func (qq *Queues) start(handlers map[string]dispatcher.HandlerFunc) error {
	err := qq.initQueues(handlers)
	if err != nil {
		return err
	}
	qq.m.Range(func(qname, q any) bool {
		q.(*Queue).start()
		logq.Log.Infof("DBQueue[%s] start", qname)
		return true
	})

	if qq.aggregator != nil {
		qq.aggregator.Start()
	}

	return nil
}

func NewQueues(ctx context.Context, conf *Conf) *Queues {
	if dbQueues == nil {
		dbQueues = &Queues{
			ctx: ctx,
			c:   conf,
		}
	}
	return dbQueues
}
