package helper

import (
	"context"
	"sync"
)

type ctxKey string

const mapKey ctxKey = "concurrent_map[string]*string"

type ctxConcurrentMap struct {
	m  sync.Map
	mu sync.RWMutex
}

// CtxWithMap 创建一个新的并发安全 map 并注入 context
func CtxWithMap(ctx context.Context) context.Context {
	return context.WithValue(ctx, mapKey, &ctxConcurrentMap{})
}

// CtxWithValue 并发安全地设置/更新 key-value
func CtxWithValue(ctx context.Context, key string, value *string) context.Context {
	cm := ctxGetConcurrentMap(ctx)
	if cm == nil {
		cm = &ctxConcurrentMap{}
		ctx = context.WithValue(ctx, mapKey, cm)
	}
	cm.m.Store(key, value)
	return ctx
}

// CtxGet 获取值
func CtxGet(ctx context.Context, key string) (*string, bool) {
	cm := ctxGetConcurrentMap(ctx)
	if cm == nil {
		return nil, false
	}
	v, ok := cm.m.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*string), true
}

// CtxMustGet 获取值，不存在返回空字符串
func CtxMustGet(ctx context.Context, key string) *string {
	v, _ := CtxGet(ctx, key)
	return v
}

// CtxDelete 删除 key
func CtxDelete(ctx context.Context, key string) {
	if cm := ctxGetConcurrentMap(ctx); cm != nil {
		cm.m.Delete(key)
	}
}

// CtxSetMap 批量替换整个 map（原子性）
func CtxSetMap(ctx context.Context, newMap map[string]*string) context.Context {
	cm := ctxGetConcurrentMap(ctx)
	if cm == nil {
		cm = &ctxConcurrentMap{}
		ctx = context.WithValue(ctx, mapKey, cm)
	}
	// 清空旧数据
	cm.m.Range(func(key, _ any) bool {
		cm.m.Delete(key)
		return true
	})
	// 写入新数据
	for k, v := range newMap {
		cm.m.Store(k, v)
	}
	return ctx
}

// CtxGetAll 获取当前所有键值对的副本（安全）
func CtxGetAll(ctx context.Context) map[string]*string {
	cm := ctxGetConcurrentMap(ctx)
	if cm == nil {
		return make(map[string]*string)
	}

	result := make(map[string]*string)
	cm.m.Range(func(key, value any) bool {
		result[key.(string)] = value.(*string)
		return true
	})
	return result
}

// ctxGetConcurrentMap 内部获取
func ctxGetConcurrentMap(ctx context.Context) *ctxConcurrentMap {
	if ctx == nil {
		return nil
	}
	if cm, ok := ctx.Value(mapKey).(*ctxConcurrentMap); ok {
		return cm
	}
	return nil
}
