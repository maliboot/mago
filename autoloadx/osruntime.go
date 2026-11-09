package autoloadx

import (
	"context"
	"fmt"
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/copool"
	"github.com/maliboot/mago/helper"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/go-co-op/gocron/v2"
)

type OSRuntime struct {
	conf        *config.Conf
	coPool      *copool.CoPool
	memoryLimit uint64

	memSamples []metrics.Sample
	cleanUp    func(ctx context.Context)
}

func NewOSRuntime(conf *config.Conf, coPool *copool.CoPool, cleanUp func(ctx context.Context)) *OSRuntime {
	ins := &OSRuntime{
		conf:        conf,
		coPool:      coPool,
		memoryLimit: conf.Server.MemoryLimit * 1024 * 1024,
		cleanUp:     cleanUp,
	}
	ins.initialize()
	return ins
}

func (o *OSRuntime) initialize() {
	if o.memoryLimit == 0 {
		sysMem, err := helper.GetSysMemory()
		if err != nil {
			hlog.Fatalf("自动计算系统内存失败.1: %v", err)
		}
		o.memoryLimit = sysMem.Available
	}
	o.memSamples = []metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/memory/classes/heap/unused:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
		{Name: "/memory/classes/os-stacks:bytes"},
		{Name: "/memory/classes/total:bytes"}, // runtime 总内存占用（近似 Sys）
		{Name: "/gc/cycles/total:gc-cycles"},
	}
}

func (o *OSRuntime) Monitor() error {
	cron, err := o.conf.Scheduler.Cron()
	if err != nil {
		return err
	}

	_, err = cron.NewJob(
		gocron.DurationJob(time.Second*60),
		gocron.NewTask(o.monitor),
		gocron.WithSingletonMode(gocron.LimitModeWait),
		gocron.WithName("OSRuntime"),
		gocron.WithTags("OSRuntime"),
	)
	return err
}

// 后台监控goroutine
func (o *OSRuntime) monitor() {
	metrics.Read(o.memSamples)
	heap := o.memSamples[0].Value.Uint64()
	heapUse := o.memSamples[1].Value.Uint64() + heap
	heapUnused := o.memSamples[2].Value.Uint64()
	heapIdle := o.memSamples[3].Value.Uint64() + o.memSamples[4].Value.Uint64() + heapUnused
	stacks := o.memSamples[5].Value.Uint64()
	currentMem := o.memSamples[6].Value.Uint64()
	totalNumGC := o.memSamples[7].Value.Uint64()
	numGoroutines := uint64(runtime.NumGoroutine())

	// 池状态
	cpuPool := o.coPool.GetCpuPool()
	ioPool := o.coPool.GetIoPool()
	heavyIoPool := o.coPool.GetHeavyIoPool()
	poolMsg := fmt.Sprintf("cpu池[运行中: %d, 空闲: %d, 等待: %d]; io池[运行中: %d, 空闲: %d, 等待: %d]; 大io池[运行中: %d, 空闲: %d, 等待: %d]",
		cpuPool.Running(), cpuPool.Free(), cpuPool.Waiting(),
		ioPool.Running(), ioPool.Free(), ioPool.Waiting(),
		heavyIoPool.Running(), heavyIoPool.Free(), heavyIoPool.Waiting(),
	)

	baseMsg := fmt.Sprintf(
		"Monitor CoNum: %d; TotalAlloc: %.2fMB(Threshold: %.2fMB)",
		numGoroutines,
		helper.BytesToMB(currentMem),
		helper.BytesToMB(o.memoryLimit),
	)
	memMsg := fmt.Sprintf(
		"Monitor MemStats: HeapAlloc=%.2fMB StacksAlloc=%.2fMB TotalAlloc=%.2fMB HeapInuse=%.2fMB HeapIdle=%.2fMB NumGC=%d",
		helper.BytesToMB(heap),
		helper.BytesToMB(stacks),
		helper.BytesToMB(currentMem),
		helper.BytesToMB(heapUse),
		helper.BytesToMB(heapIdle),
		totalNumGC,
	)

	// error
	errMemLimit := o.memoryLimit * 9 / 10
	if currentMem > errMemLimit {
		hlog.Errorf(baseMsg)
		hlog.Errorf(memMsg)
		hlog.Errorf(poolMsg)
		hlog.Errorf("Monitor panic: [%s]", helper.AesEncryptLog(debug.Stack(), ""))

		if o.cleanUp != nil {
			o.cleanUp(context.Background())
		}

		time.Sleep(time.Second * 5)
		if err := helper.SendSelfSignal(syscall.SIGTERM); err != nil {
			hlog.Errorf("OSRuntime Failed to send SIGTERM: %v", err)
			os.Exit(500)
		}

		hlog.Errorf("main.服务退出")
		return
	}

	// warn
	memWarnLimit := o.memoryLimit * 8 / 10
	if currentMem > memWarnLimit {
		hlog.Warnf(baseMsg)
		hlog.Warnf(memMsg)
		hlog.Warnf(poolMsg)
		// 触发GC以尝试缓解
		runtime.GC()
		return
	}

	now := time.Now()
	// 整点报数
	if now.Minute() == 0 && now.Second() < 60 {
		hlog.Infof(baseMsg)
		hlog.Infof(memMsg)
		hlog.Infof(poolMsg)
	}
}

func (o *OSRuntime) currentMemory() uint64 {
	metrics.Read(o.memSamples)
	heap := o.memSamples[0].Value.Uint64()
	free := o.memSamples[1].Value.Uint64()
	stacks := o.memSamples[2].Value.Uint64()
	return heap + stacks - free
}
