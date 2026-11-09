package config

import (
	"github.com/maliboot/mago/helper"
	"runtime/debug"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

type Scheduler struct {
	LimitConcurrentNum uint `yaml:"limit_concurrent_num"`
	Ctx                any

	cron gocron.Scheduler
}

func (r *Scheduler) Cron() (gocron.Scheduler, error) {
	if r.cron == nil {
		var err error
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.Local
		}
		r.cron, err = gocron.NewScheduler(
			gocron.WithLocation(loc),
			gocron.WithLogger(&SchedulerLogger{}),
			gocron.WithGlobalJobOptions(
				gocron.WithEventListeners(
					r.globalAfterJobRunsWithError(),
					r.globalAfterJobRunsWithFatal(),
				), // 全局事件监听
			),
		)
		if err != nil {
			return nil, err
		}
	}
	return r.cron, nil
}

func (r *Scheduler) globalBeforeJobRuns() gocron.EventListener {
	return gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
		hlog.Debugf("[%s]定时任务[%s]执行开始", jobName, jobID)
	})
}

func (r *Scheduler) globalAfterJobRuns() gocron.EventListener {
	return gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
		hlog.Debugf("[%s]定时任务[%s]执行结束", jobName, jobID)
	})
}

func (r *Scheduler) globalAfterJobRunsWithError() gocron.EventListener {
	return gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
		hlog.Errorf("[%s]定时任务[%s]执行异常: %v", jobName, jobID, err)
	})
}

func (r *Scheduler) globalAfterJobRunsWithFatal() gocron.EventListener {
	return gocron.AfterJobRunsWithPanic(func(jobID uuid.UUID, jobName string, recoverData any) {
		hlog.Errorf("[%s]定时任务[%s]执行Painc: [%v], stack[%s]", jobName, jobID, recoverData, helper.AesEncryptLog(debug.Stack(), ""))
	})
}

type SchedulerLogger struct {
}

func (l *SchedulerLogger) Debug(msg string, args ...any) {
	hlog.Debug(msg, args)
}

func (l *SchedulerLogger) Error(msg string, args ...any) {
	hlog.Error(msg, args)
}

func (l *SchedulerLogger) Info(msg string, args ...any) {
	hlog.Info(msg, args)
}

func (l *SchedulerLogger) Warn(msg string, args ...any) {
	hlog.Warn(msg, args)
}
