package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/btree-query-bench/bmark/index"
	"github.com/btree-query-bench/bmark/index/bplustree"
	"github.com/btree-query-bench/bmark/index/btree"
	"github.com/btree-query-bench/bmark/index/lsmtree"
)

func main() {
	f, _ := os.Create("final_thesis_results.csv")
	defer f.Close()
	w := csv.NewWriter(f)
	// Added HeapObjects to the header to track GC pressure
	w.Write([]string{"Structure", "Config", "TestType", "LatencyNs", "MemMB", "HeapObjects"})

	// Configuration arrays
	degrees := []int{8, 32, 128}
	lsmThresholds := []int{1000, 10000}

	scale := 1000000

	// --- 1. Sweep B-Tree & B+Tree ---
	for _, d := range degrees {
		runSuite(w, "B-Tree", d, btree.NewBTree(d), scale)
		runSuite(w, "BPlusTree", d, bplustree.NewBPlusTree(d), scale)
	}

	// --- 2. Sweep LSM ---
	for _, t := range lsmThresholds {
		runSuite(w, "LSM-Tree", t, lsmtree.NewLSM(t), scale)
	}

	w.Flush()
	fmt.Println("Benchmark complete. Data ready for analysis.")
}

func runSuite(w *csv.Writer, name string, conf int, idx interface{}, n int) {
	fmt.Printf("Testing %s (Config: %d)\n", name, conf)
	confStr := strconv.Itoa(conf)

	var i index.Index
	switch v := idx.(type) {
	case *btree.BTree:
		i = v
	case *bplustree.BPlusTree:
		i = v
	case *lsmtree.LSMTree:
		i = v
	}

	// 1. Pure Insert (Initial Load)
	start := time.Now()
	for k := 0; k < n; k++ {
		i.Insert(int64(k), []byte("v"))
	}
	insertLatency := time.Since(start).Nanoseconds() / int64(n)

	// --- MEMORY FOOTPRINT SAMPLING ---
	// Measure memory immediately after load but before workloads
	stats := GetDetailedMem()
	Record(w, BenchResult{
		Name:      name,
		Config:    confStr,
		Operation: "Footprint_SteadyState",
		LatencyNs: insertLatency,
		MemMB:     stats.AllocMB,
		Objects:   stats.HeapObjects,
	})

	// 2. Scenario: OLTP (Read Heavy)
	start = time.Now()
	ExecuteWorkload(i, OLTP, n/2)
	Record(w, BenchResult{name, confStr, "Workload_OLTP", time.Since(start).Nanoseconds() / int64(n/2), GetDetailedMem().AllocMB, 0})

	// 3. Scenario: OLAP (Write Heavy)
	start = time.Now()
	ExecuteWorkload(i, OLAP, n/2)
	Record(w, BenchResult{name, confStr, "Workload_OLAP", time.Since(start).Nanoseconds() / int64(n/2), GetDetailedMem().AllocMB, 0})

	// 4. Basic: Range Scan
	start = time.Now()
	ExecuteWorkload(i, Reporting, 100)
	Record(w, BenchResult{name, confStr, "Workload_Range", time.Since(start).Nanoseconds() / 100, GetDetailedMem().AllocMB, 0})
}
