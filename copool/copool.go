package copool

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"github.com/maliboot/mago/queuex/archiveq"
	"io"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/panjf2000/ants/v2"

	econfig "github.com/maliboot/mago/config"
	"github.com/maliboot/mago/dispatcher"
	"github.com/maliboot/mago/emqx"
	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/httpx"
	"github.com/maliboot/mago/queuex"
)

type CoPool struct {
	conf   *econfig.Conf
	ctx    context.Context
	cancel context.CancelFunc

	computePool                  *ants.Pool
	computePoolMaxSize           int
	computePoolBlockQueueMaxSize int
	ioPool                       *ants.Pool
	ioPoolMaxSize                int
	heavyIoPool                  *ants.Pool
	heavyIoPoolMaxSize           int
	syncPool                     *eghelper.SyncBuf

	httpIns *httpx.Client
	mqttIns *emqx.MQTT

	queueClt queuex.Client
	queueSrv queuex.Server
}

func NewCoPool(conf *econfig.Conf, queueSrv queuex.Server, mqttIns *emqx.MQTT) *CoPool {
	numCPU := runtime.NumCPU()
	totalMemMB := conf.Server.MemoryLimit
	if totalMemMB == 0 {
		// 自动读取总容量
		sysMem, err := eghelper.GetSysMemory()
		if err != nil {
			hlog.Fatalf("自动计算系统内存失败: %v", err)
		}
		totalMemMB = sysMem.Available / 1024 / 1024
	}

	usableMem := totalMemMB * 1024 * 1024 / 3 // 分3分之1内存给所有池子
	safetyFactor := 0.5                       // 安全系数，防止全部可用内存都被任务占满，留出 GC、sqlite 等额外开销
	// 分配比例：cpu:小IO:大IO = 4:4:2
	cpuRatio := 0.4
	normalIoRatio := 0.4
	heavyIoRatio := 0.2

	// ---------------- CPU池 ----------------
	// CPU 密集任务 = CPU核数 × 1~4
	cpuPoolLimit := numCPU * 4
	if cpuPoolLimit > 64 {
		cpuPoolLimit = 64 // 上限保护
	}
	// 32KB buffer
	cpuBuf := 1024 * 32
	cpuPoolMaxLimit := int(float64(usableMem) / float64(cpuBuf) * cpuRatio * safetyFactor)
	cpuPoolBlockQueueLimit := cpuPoolMaxLimit - cpuPoolLimit
	if cpuPoolBlockQueueLimit < 0 {
		cpuPoolBlockQueueLimit = 0
	}

	// -------- IO池 --------
	// 64KB buffer
	normalBuf := 64 * 1024
	normalIoPoolLimit := int(float64(usableMem) / float64(normalBuf) * normalIoRatio * safetyFactor)
	// 2MB buffer
	heavyBuf := 2 * 1024 * 1024
	heavyIoPoolLimit := int(float64(usableMem) / float64(heavyBuf) * heavyIoRatio * safetyFactor)

	// -------- 限制保护 --------
	if normalIoPoolLimit < 100 {
		normalIoPoolLimit = 100
	}
	if heavyIoPoolLimit < 10 {
		heavyIoPoolLimit = 10
	}

	maxIo := numCPU * 500
	if normalIoPoolLimit > maxIo {
		normalIoPoolLimit = maxIo
	}
	maxHeavy := numCPU * 200
	if heavyIoPoolLimit > maxHeavy {
		heavyIoPoolLimit = maxHeavy
	}

	ins := &CoPool{
		conf:                         conf,
		queueClt:                     conf.Queue.MustGetClient(),
		queueSrv:                     queueSrv,
		mqttIns:                      mqttIns,
		computePoolMaxSize:           cpuPoolLimit,
		computePoolBlockQueueMaxSize: cpuPoolBlockQueueLimit,
		ioPoolMaxSize:                normalIoPoolLimit,
		heavyIoPoolMaxSize:           heavyIoPoolLimit,
		syncPool:                     eghelper.NewSyncBuf(),
	}

	hlog.Infof(
		"协程池初始化完成: 检测到可用总内存[%dMB], 分配置池总内存[%.2fMB], cpu池[%d+%d], io池[%d], 大io池[%d]",
		totalMemMB, float64(usableMem)/1024/1024, cpuPoolLimit, cpuPoolBlockQueueLimit, normalIoPoolLimit, heavyIoPoolLimit,
	)
	return ins
}

func (o *CoPool) initialize() {
	// 计算密集任务池
	o.computePool, _ = ants.NewPool(
		o.computePoolMaxSize,
		ants.WithPreAlloc(true),
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(o.computePoolBlockQueueMaxSize),
		ants.WithPanicHandler(o.poolPanic),
	)

	// IO密集任务池
	o.ioPool, _ = ants.NewPool(
		o.ioPoolMaxSize,
		ants.WithPreAlloc(false),
		ants.WithNonblocking(true),
		ants.WithMaxBlockingTasks(0),
		ants.WithExpiryDuration(10*time.Second),
		ants.WithPanicHandler(o.poolPanic),
	)

	// 大IO密集任务池
	o.heavyIoPool, _ = ants.NewPool(
		o.heavyIoPoolMaxSize,
		ants.WithPreAlloc(false),
		ants.WithNonblocking(true),
		ants.WithMaxBlockingTasks(0),
		ants.WithExpiryDuration(10*time.Second),
		ants.WithPanicHandler(o.poolPanic),
	)

	// 全局httpClient
	o.httpIns = httpx.NewClient(o.syncPool)
}

func (o *CoPool) GetContext() context.Context {
	return o.ctx
}

func (o *CoPool) poolPanic(err any) {
	hlog.Errorf("copool panic: [%v][%s]", err, eghelper.AesEncryptLog(debug.Stack(), ""))
}

func (o *CoPool) cronPanic(name string, err any) {
	hlog.Errorf("cron[%s] panic: [%v][%s]", name, err, eghelper.AesEncryptLog(debug.Stack(), ""))
}

func (o *CoPool) Start(ctx context.Context) error {
	o.ctx, o.cancel = context.WithCancel(ctx)
	o.initialize()

	hlog.Infof("协程池准备就绪✅✅✅")
	return nil
}

func (o *CoPool) Stop(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	o.Release()
	hlog.Infof("协程池退出完成✅✅✅")
	return nil
}

func (o *CoPool) Release() {
	if o.computePool != nil {
		o.computePool.Release()
	}
	if o.ioPool != nil {
		o.ioPool.Release()
	}
	if o.heavyIoPool != nil {
		o.heavyIoPool.Release()
	}
}

func (o *CoPool) GetSyncPool() *eghelper.SyncBuf {
	return o.syncPool
}

func (o *CoPool) GetCpuPool() *ants.Pool {
	return o.computePool
}

func (o *CoPool) GetIoPool() *ants.Pool {
	return o.ioPool
}

func (o *CoPool) GetHeavyIoPool() *ants.Pool {
	return o.heavyIoPool
}

func (o *CoPool) GetIoPoolIdleRate() float64 {
	ioCap := o.ioPool.Cap()
	if ioCap <= 0 {
		return 0.0 // 动态容量或默认池特殊处理
	}
	ioFree := o.ioPool.Free()
	return float64(ioFree) / float64(ioCap)
}

// SubmitComputeTask 提交计算密集型任务
func (o *CoPool) SubmitComputeTask(taskNo string, task func()) error {
	err := o.computePool.Submit(task)
	if err != nil {
		hlog.Errorf("cpu协程池[%s]投递失败: %v", taskNo, err)
	}
	return err
}

// SubmitIOTask 提交IO密集型任务
func (o *CoPool) SubmitIOTask(taskNo string, task func()) error {
	err := o.ioPool.Submit(task)
	if err != nil {
		hlog.Errorf("io协程池[%s]投递失败: %v", taskNo, err)
	}
	return err
}

// SubmitQueueTask 提交队列任务-排队控制
func (o *CoPool) SubmitQueueTask(taskData *dispatcher.Task) error {
	_, err := o.queueClt.Enqueue(taskData)
	if err != nil {
		hlog.Errorf("queue投递失败[%+v], err=%v", taskData.Headers, err)
	}
	return err
}

// RegisterQueueTaskWorker
// 队列启动前配置，否则无效.
// @timeout仅仅对db队列有效.
func (o *CoPool) RegisterQueueTaskWorker(queueName string, weight int, timeout time.Duration) {
	o.queueSrv.RegisterQueues(map[string]int{queueName: weight})
	o.queueSrv.RegisterQueuesTimeout(map[string]time.Duration{queueName: timeout})
}

// RegisterQueueTaskHandler
// 队列启动前配置，否则无效
func (o *CoPool) RegisterQueueTaskHandler(taskName string, handlerFn dispatcher.HandlerFunc) {
	o.queueSrv.RegisterHandleFunc(taskName, handlerFn)
}

func (o *CoPool) RegisterQueueTaskAggregatorHandler(group string, handlerFunc dispatcher.AggregatorHandlerFunc) {
	o.queueSrv.RegisterAggregatorHandler(group, handlerFunc)
}

func (o *CoPool) QueueArchiverGet(ctx context.Context, isSuccess bool, page, pageSize int, ext map[string]string) ([]*archiveq.ArchiverMessage, error) {
	return o.queueSrv.ArchiverGet(ctx, isSuccess, page, pageSize, ext)
}
func (o *CoPool) QueueArchiverFlush(isSuccess bool) int {
	return o.queueSrv.ArchiverFlush(isSuccess)
}

func (o *CoPool) getPoolByTaskType(taskType dispatcher.Type) *ants.Pool {
	switch taskType {
	case dispatcher.CPU:
		return o.computePool
	case dispatcher.IO:
		return o.ioPool
	case dispatcher.HeavyIO:
		return o.heavyIoPool
	default:
		return o.ioPool
	}
}

func (o *CoPool) MqttRequestTask(topic string, payload []byte) bool {
	if topic == "" || o.mqttIns == nil || !o.mqttIns.GetStatus() {
		hlog.Errorf("MqttRequestTaskErr.1:[%s]", base64.StdEncoding.EncodeToString(payload))
		return false
	}

	err := o.ioPool.Submit(func() {
		ret := o.mqttIns.Publish(topic, 2, false, payload)
		if !ret {
			hlog.Errorf("MqttRequestTaskErr.2:[%s]", base64.StdEncoding.EncodeToString(payload))
		}
	})

	return err == nil
}

func (o *CoPool) HttpPostAsync(httpUrl string, headers map[string]string, body []byte, timeout time.Duration, cb func(*protocol.Response, error)) error {
	err := o.SubmitIOTask(httpUrl, func() {
		res, err := o.httpIns.Post(httpUrl, headers, body, timeout)
		defer func() {
			if res != nil {
				res.Reset()
			}
		}()
		if cb != nil {
			cb(res, err)
		}
	})
	if err != nil {
		hlog.Errorf("CoPool_HttpPostAsyncErr:%v", err)
	}
	return err
}

func (o *CoPool) HttpPost(httpUrl string, headers map[string]string, body []byte, timeout time.Duration, cb func(*protocol.Response, error) error) error {
	res, err := o.httpIns.Post(httpUrl, headers, body, timeout)
	defer func() {
		if res != nil {
			res.Reset()
		}
	}()
	if cb != nil {
		return cb(res, err)
	}
	return err
}

func (o *CoPool) Http2Get(httpUrl string, headers map[string]string, timeout time.Duration, maxRedirectsCount int, cb func(*protocol.Response, error) error) error {
	res, err := o.httpIns.Http2Get(httpUrl, headers, timeout, maxRedirectsCount)
	defer func() {
		if res != nil {
			res.Reset()
		}
	}()
	if cb != nil {
		return cb(res, err)
	}
	return err
}

func (o *CoPool) HttpStreamGet(
	httpUrl string,
	headers map[string]string,
	timeout time.Duration,
	maxRedirectsCount int,
	dst io.Writer,
	streamBufferLimit uint,
	cb func(*protocol.ResponseHeader, error) error,
) error {
	res, err := o.httpIns.StreamGet(httpUrl, headers, timeout, maxRedirectsCount, dst, streamBufferLimit)
	defer func() {
		if res != nil {
			res.Reset()
		}
	}()
	if cb != nil {
		return cb(res, err)
	}
	return err
}

func (o *CoPool) IoCron(ctx context.Context, cronName string, timeInterval time.Duration, task func(context.Context)) string {
	return o.Cron(ctx, dispatcher.IO, cronName, timeInterval, task)
}

func (o *CoPool) CpuCron(ctx context.Context, cronName string, timeInterval time.Duration, task func(context.Context)) string {
	return o.Cron(ctx, dispatcher.CPU, cronName, timeInterval, task)
}

func (o *CoPool) Cron(ctx context.Context, taskType dispatcher.Type, cronName string, timeInterval time.Duration, task func(context.Context)) string {
	jobIdBytes := md5.Sum([]byte(cronName))
	jobId := hex.EncodeToString(jobIdBytes[:])

	go func() {
		time.Sleep(time.Second)
		pool := o.getPoolByTaskType(taskType)
		ticker := time.NewTicker(timeInterval)
		hlog.Infof("定时器[%s]启动", cronName)
		defer func() {
			hlog.Infof("定时器[%s]退出", cronName)
			ticker.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-o.ctx.Done():
				return
			case <-ticker.C:
				// 将任务提交到协程池
				err := pool.Submit(func() {
					defer func() {
						if r := recover(); r != nil {
							o.cronPanic(cronName, r)
						}
						//hlog.Debugf("cron[%s]执行成功", cronName)
					}()
					task(ctx)
				})
				if err != nil {
					hlog.Errorf("定时器[%s]提交任务到协程池异常: %v", cronName, err)
				}
			}
		}
	}()

	// jobId
	return jobId
}
