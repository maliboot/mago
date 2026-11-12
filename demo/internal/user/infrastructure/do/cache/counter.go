// Package cache 存放本机内存结构（DO），按通用分类（内存/缓存）命名：
// 只描述字段，读写、并发与生命周期控制在 infrastructure/repository。
package cache

// Counter 是编号计数器的内存快照：记录当天计数所属日期与已发放序号。
type Counter struct {
	// Date 是计数所属日期，格式 yyyyMMdd。
	Date string
	// Value 是当天已发放的序号，下一个编号从这里 +1。
	Value int64
}
