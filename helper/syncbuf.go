package helper

import "sync"

const (
	normalBufPoolSize = 64 * 1024
	middleBufPoolSize = 256 * 1024
	bigBufPoolSize    = 1024 * 1024
	superBufPoolSize  = 1024 * 1024 * 2
)

type SyncBuf struct {
	// buffer池
	bufferPools []sync.Pool
}

func NewSyncBuf() *SyncBuf {
	ins := &SyncBuf{bufferPools: []sync.Pool{
		{New: func() interface{} { return make([]byte, normalBufPoolSize) }},
		{New: func() interface{} { return make([]byte, middleBufPoolSize) }},
		{New: func() interface{} { return make([]byte, bigBufPoolSize) }},
		{New: func() interface{} { return make([]byte, superBufPoolSize) }},
	}}

	return ins
}

func (g *SyncBuf) Get(size int64) []byte {
	switch {
	case size <= normalBufPoolSize:
		return g.bufferPools[0].Get().([]byte)[:size]
	case size <= middleBufPoolSize:
		return g.bufferPools[1].Get().([]byte)[:size]
	case size <= bigBufPoolSize:
		return g.bufferPools[2].Get().([]byte)[:size]
	case size <= superBufPoolSize:
		return g.bufferPools[3].Get().([]byte)[:size]
	default:
		return make([]byte, size) //不使用池
	}
}

func (g *SyncBuf) Put(buf []byte) {
	switch cap(buf) {
	case normalBufPoolSize:
		g.bufferPools[0].Put(buf)
	case middleBufPoolSize:
		g.bufferPools[1].Put(buf)
	case bigBufPoolSize:
		g.bufferPools[2].Put(buf)
	case superBufPoolSize:
		g.bufferPools[3].Put(buf)
	}
}
