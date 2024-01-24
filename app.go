package mago

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"golang.org/x/sync/errgroup"
)

type App struct {
	Name        string
	ctx         context.Context
	cancel      context.CancelFunc
	Servers     []Server
	stopTimeout time.Duration
	stopErr     error
}

type StopError struct {
}

func New(name string, servers []Server) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		Name:        name,
		Servers:     servers,
		ctx:         ctx,
		cancel:      cancel,
		stopTimeout: time.Second * 5,
	}
}

func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.ctx)
	wg := sync.WaitGroup{}

	for _, srv := range a.Servers {
		if srv == nil {
			continue
		}
		srv := srv
		eg.Go(func() error {
			<-ctx.Done() // wait for stop signal
			stopCtx, cancel := context.WithTimeout(context.WithValue(a.ctx, StopError{}, a.stopErr), a.stopTimeout)
			defer cancel()
			return srv.Stop(stopCtx)
		})
		wg.Add(1)
		eg.Go(func() error {
			wg.Done() // here is to ensure server start has begun running before register, so defer is not needed
			return srv.Start(a.ctx)
		})
	}
	wg.Wait()

	signalToNotify := []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM}
	if signal.Ignored(syscall.SIGHUP) {
		signalToNotify = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, signalToNotify...)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-signals:
			return a.Stop(sig)
		}
	})
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (a *App) Stop(sig os.Signal) error {
	if a.cancel != nil {
		defer a.cancel()
	}

	switch sig {
	case syscall.SIGTERM:
		// force exit
		a.stopErr = errors.New(sig.String())
		return a.stopErr // nolint
	case syscall.SIGHUP, syscall.SIGINT:
		hlog.SystemLogger().Infof("Received signal: %s\n", sig)
		// graceful shutdown
		return nil
	}

	return nil
}
