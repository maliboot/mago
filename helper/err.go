package helper

import (
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func PanicRecover(name string) {
	if r := recover(); r != nil {
		hlog.Errorf("[%s]panic: [%v][%s]", name, r, AesEncryptLog(debug.Stack(), ""))
	}
}
