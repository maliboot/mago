package mago

import (
	"context"
	"errors"
	"fmt"
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

	var startErrMu sync.Mutex
	var firstStartErr error
	for _, srv := range a.Servers {
		if srv == nil {
			continue
		}
		srv := srv
		eg.Go(func() error {
			startDone := make(chan error, 1)
			go func() {
				err := srv.Start(a.ctx)
				startDone <- err
			}()

			select {
			case err := <-startDone:
				if err != nil {
					hlog.Errorf("%s server [%T] start error: %v", a.Name, srv, err)
					startErrMu.Lock()
					if firstStartErr == nil {
						firstStartErr = err
					}
					startErrMu.Unlock()
					return err
				}
				<-ctx.Done()
			case <-ctx.Done():
				// 在 Start 还没完成时就收到停止信号（极少发生，但要处理）
				return nil
			}

			stopCtx, cancel := context.WithTimeout(context.WithValue(a.ctx, StopError{}, a.stopErr), a.stopTimeout)
			defer cancel()

			if err := srv.Stop(stopCtx); err != nil {
				hlog.Errorf("%s server stop error: %v", a.Name, err)
				return err
			}
			return nil
		})
	}

	eg.Go(func() error {
		signalToNotify := []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM}
		if signal.Ignored(syscall.SIGHUP) {
			signalToNotify = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
		}
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, signalToNotify...)

		select {
		case <-ctx.Done():
			return nil
		case sig := <-signals:
			return a.Stop(sig)
		}
	})
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		if firstStartErr != nil {
			return firstStartErr
		}
		return err
	}
	return nil
}

func (a *App) Stop(sig os.Signal) error {
	if a.stopErr == nil {
		a.stopErr = fmt.Errorf("received signal: %v", sig)
	}

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
