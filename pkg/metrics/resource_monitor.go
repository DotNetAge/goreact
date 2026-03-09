package metrics

import (
	"runtime"
	"sync"
	"time"
)

// ResourceMonitor 系统资源监控器
type ResourceMonitor struct {
	startMemStats runtime.MemStats
	startTime     time.Time
	mutex         sync.Mutex
}

// NewResourceMonitor 创建新的资源监控器
func NewResourceMonitor() *ResourceMonitor {
	monitor := &ResourceMonitor{
		startTime: time.Now(),
	}
	runtime.ReadMemStats(&monitor.startMemStats)
	return monitor
}

// Snapshot 获取当前资源使用快照
func (rm *ResourceMonitor) Snapshot() *ResourceSnapshot {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &ResourceSnapshot{
		Timestamp:     time.Now(),
		MemoryAllocMB: float64(memStats.Alloc) / 1024 / 1024,
		MemoryTotalMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		MemorySysMB:   float64(memStats.Sys) / 1024 / 1024,
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		GCPauseMs:     float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e6,
		NumGC:         memStats.NumGC,
		HeapAllocMB:   float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:     float64(memStats.HeapSys) / 1024 / 1024,
		HeapIdleMB:    float64(memStats.HeapIdle) / 1024 / 1024,
		HeapInUseMB:   float64(memStats.HeapInuse) / 1024 / 1024,
		StackInUseMB:  float64(memStats.StackInuse) / 1024 / 1024,
	}
}

// ResourceSnapshot 资源使用快照
type ResourceSnapshot struct {
	Timestamp     time.Time
	MemoryAllocMB float64 // 当前分配的内存 (MB)
	MemoryTotalMB float64 // 累计分配的内存 (MB)
	MemorySysMB   float64 // 从系统获取的内存 (MB)
	NumGoroutines int     // Goroutine 数量
	NumCPU        int     // CPU 核心数
	GCPauseMs     float64 // 最近一次 GC 暂停时间 (ms)
	NumGC         uint32  // GC 次数
	HeapAllocMB   float64 // 堆分配的内存 (MB)
	HeapSysMB     float64 // 堆从系统获取的内存 (MB)
	HeapIdleMB    float64 // 堆空闲内存 (MB)
	HeapInUseMB   float64 // 堆正在使用的内存 (MB)
	StackInUseMB  float64 // 栈使用的内存 (MB)
}

// Delta 计算两个快照之间的差异
func (rs *ResourceSnapshot) Delta(before *ResourceSnapshot) *ResourceDelta {
	return &ResourceDelta{
		Duration:      rs.Timestamp.Sub(before.Timestamp),
		MemoryAllocMB: rs.MemoryAllocMB - before.MemoryAllocMB,
		MemoryTotalMB: rs.MemoryTotalMB - before.MemoryTotalMB,
		NumGoroutines: rs.NumGoroutines - before.NumGoroutines,
		GCCount:       int(rs.NumGC - before.NumGC),
		HeapAllocMB:   rs.HeapAllocMB - before.HeapAllocMB,
	}
}

// ResourceDelta 资源使用变化
type ResourceDelta struct {
	Duration      time.Duration
	MemoryAllocMB float64 // 内存分配变化 (MB)
	MemoryTotalMB float64 // 累计内存分配变化 (MB)
	NumGoroutines int     // Goroutine 数量变化
	GCCount       int     // GC 次数变化
	HeapAllocMB   float64 // 堆内存分配变化 (MB)
}
