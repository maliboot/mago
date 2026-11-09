package helper

import (
	"fmt"
	"os"
)

// SendSelfSignal 向自身发送信号
func SendSelfSignal(signal os.Signal) error {
	// 获取当前进程
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}
	// 发送信号
	if err := p.Signal(signal); err != nil {
		return fmt.Errorf("failed to send signal %v: %v", signal, err)
	}
	return nil
}
