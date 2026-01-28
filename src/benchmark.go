package main

import (
	"encoding/csv"
	"runtime"
	"strconv"
)

// BenchResult updated to include Objects for GC pressure analysis
type BenchResult struct {
	Name      string
	Config    string
	Operation string
	LatencyNs int64
	MemMB     uint64
	Objects   uint64
}

type MemoryStats struct {
	AllocMB      uint64
	TotalAllocMB uint64
	HeapObjects  uint64
}

// GetDetailedMem replaces GetMem for better academic data
func GetDetailedMem() MemoryStats {
	var m runtime.MemStats
	// Force GC to ensure we measure actual live data, not garbage
	runtime.GC()
	runtime.ReadMemStats(&m)
	return MemoryStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		HeapObjects:  m.HeapObjects,
	}
}

// Record now writes 6 columns to your CSV
func Record(w *csv.Writer, res BenchResult) {
	w.Write([]string{
		res.Name,
		res.Config,
		res.Operation,
		strconv.FormatInt(res.LatencyNs, 10),
		strconv.FormatUint(res.MemMB, 10),
		strconv.FormatUint(res.Objects, 10),
	})
}
