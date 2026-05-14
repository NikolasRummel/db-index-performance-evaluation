package bench

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/btree-query-bench/bmark/dbms/index"
)

type T1Result struct {
	Index     string
	NDataset  int
	NQueries  int
	MinNs     int64
	Q1Ns      int64
	P50Ns     int64
	Q3Ns      int64
	MaxNs     int64
	AvgNs     int64
	P95Ns     int64
	P99Ns     int64
	OpsPerSec float64
	TotalMs   int64
}

var t1Header = []string{
	"index", "n_dataset", "n_queries",
	"min_ns", "q1_ns", "p50_ns", "q3_ns", "max_ns",
	"avg_ns", "p95_ns", "p99_ns",
	"ops_per_sec", "total_ms",
}

func fillIndex(idx index.Index, ds Dataset) error {
	// Disable sync for initial fill to speed up preparation.
	if s, ok := idx.(interface{ SetSyncInterval(int) }); ok {
		s.SetSyncInterval(0)
	}

	// Create a slice of indices and sort it based on the keys.
	// This allows us to insert keys in ascending order, which is much faster
	// for B-tree and B+ tree structures as it minimizes splits and re-balancing.
	indices := make([]int, len(ds.Keys))
	for i := range indices {
		indices[i] = i
	}

	sort.Slice(indices, func(i, j int) bool {
		return ds.Keys[indices[i]] < ds.Keys[indices[j]]
	})

	for _, i := range indices {
		k := ds.Keys[i]
		if err := idx.Insert(k, ds.Values[i]); err != nil {
			return fmt.Errorf("insert key %d: %w", k, err)
		}
	}
	return nil
}

func cleanupIndexData(path string) {
	// Try to remove as a directory (LSM)
	_ = os.RemoveAll(path)
	// Try to remove as a B-tree file
	_ = os.Remove(path + ".bt")
	// Try to remove as a B+ tree file
	_ = os.Remove(path + ".bpt")
}

// RunBenchmarkT1 executes the point query benchmark (T1).
// It fills each index with a dataset and measures the response time and throughput of random point queries.
func RunBenchmarkT1(indices []IndexDef, cfg Config) error {
	ds := NewDataset(cfg.DatasetSize, cfg.ValueSize, cfg.Seed)
	queryKeys := ds.RandomKeys(cfg.PointQueryCount)

	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}
	f, err := os.Create(filepath.Join(cfg.OutDir, "t1_point_query.csv"))
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write(t1Header)

	for _, def := range indices {
		fmt.Printf("[T1] %s: filling index with %d keys...\n", def.Name, cfg.DatasetSize)

		idxPath := filepath.Join(cfg.DataDir, def.Name+"_t1")
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			fmt.Printf("[T1] %s: open failed: %v — skipping\n", def.Name, err)
			continue
		}

		if err := fillIndex(idx, *ds); err != nil {
			fmt.Printf("[T1] %s: fill failed: %v — skipping\n", def.Name, err)
			_ = idx.Close()
			if cfg.CleanupData {
				cleanupIndexData(idxPath)
			}
			continue
		}

		if h, ok := idx.(interface{ Height() int }); ok {
			fmt.Printf("[T1] %s: tree height = %d\n", def.Name, h.Height())
		}
		if l, ok := idx.(interface{ Levels() string }); ok {
			fmt.Printf("[T1] %s: lsm levels = %s\n", def.Name, l.Levels())
		}

		fmt.Printf("[T1] %s: running %d point queries...\n", def.Name, cfg.PointQueryCount)

		responetimes := make([]int64, 0, cfg.PointQueryCount)
		start := time.Now()

		for _, key := range queryKeys {
			t := time.Now()
			val, e := idx.Get(key)
			responetimes = append(responetimes, time.Since(t).Nanoseconds())
			if e != nil {
				fmt.Printf("[T1] %s: Get(%d) error: %v\n", def.Name, key, e)
			} else if val == nil {
				fmt.Printf("[T1] %s: key %d not found\n", def.Name, key)
			}
		}

		totalDuration := time.Since(start)
		_ = idx.Close()

		if cfg.CleanupData {
			cleanupIndexData(idxPath)
		}

		sort.Slice(responetimes, func(i, j int) bool { return responetimes[i] < responetimes[j] })

		r := T1Result{
			Index:     def.Name,
			NDataset:  cfg.DatasetSize,
			NQueries:  cfg.PointQueryCount,
			MinNs:     responetimes[0],
			Q1Ns:      pct(responetimes, 25),
			P50Ns:     pct(responetimes, 50),
			Q3Ns:      pct(responetimes, 75),
			MaxNs:     responetimes[len(responetimes)-1],
			AvgNs:     avg(responetimes),
			P95Ns:     pct(responetimes, 95),
			P99Ns:     pct(responetimes, 99),
			OpsPerSec: float64(cfg.PointQueryCount) / totalDuration.Seconds(),
			TotalMs:   totalDuration.Milliseconds(),
		}

		fmt.Printf("[T1] %s: min=%dns p50=%dns avg=%dns p95=%dns p99=%dns tput=%.0f ops/s\n",
			r.Index, r.MinNs, r.P50Ns, r.AvgNs, r.P95Ns, r.P99Ns, r.OpsPerSec)

		_ = w.Write([]string{
			r.Index,
			strconv.Itoa(r.NDataset),
			strconv.Itoa(r.NQueries),
			strconv.FormatInt(r.MinNs, 10),
			strconv.FormatInt(r.Q1Ns, 10),
			strconv.FormatInt(r.P50Ns, 10),
			strconv.FormatInt(r.Q3Ns, 10),
			strconv.FormatInt(r.MaxNs, 10),
			strconv.FormatInt(r.AvgNs, 10),
			strconv.FormatInt(r.P95Ns, 10),
			strconv.FormatInt(r.P99Ns, 10),
			strconv.FormatFloat(r.OpsPerSec, 'f', 2, 64),
			strconv.FormatInt(r.TotalMs, 10),
		})
	}

	fmt.Printf("[T1] results written to %s\n", filepath.Join(cfg.OutDir, "t1_point_query.csv"))
	return nil
}

type T2Result struct {
	Index     string
	RangeSize int
	KeysRead  int
	TotalMs   int64
	OpsPerSec float64
}

var t2Header = []string{
	"index", "range_size", "keys_read",
	"total_ms", "ops_per_sec",
}

// RunBenchmarkT2 executes the range query benchmark (T2).
// It fills each index and measures the performance of scanning various range sizes.
func RunBenchmarkT2(indices []IndexDef, cfg Config) error {
	ds := NewDataset(cfg.DatasetSize, cfg.ValueSize, cfg.Seed)
	sortedKeys := ds.SortedKeys()

	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}
	f, err := os.Create(filepath.Join(cfg.OutDir, "t2_range_query.csv"))
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write(t2Header)

	for _, def := range indices {
		idxPath := filepath.Join(cfg.DataDir, def.Name+"_t2")
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			fmt.Printf("[T2] %s: open failed: %v — skipping\n", def.Name, err)
			continue
		}

		if err := fillIndex(idx, *ds); err != nil {
			fmt.Printf("[T2] %s: fill failed: %v — skipping\n", def.Name, err)
			_ = idx.Close()
			if cfg.CleanupData {
				cleanupIndexData(idxPath)
			}
			continue
		}

		if h, ok := idx.(interface{ Height() int }); ok {
			leaves := 0
			if l, ok := idx.(interface{ CountLeaves() int }); ok {
				leaves = l.CountLeaves()
			}
			fmt.Printf("[T2] %s: tree height = %d, leaves = %d\n", def.Name, h.Height(), leaves)
		}

		var t2Sizes []int
		for s := cfg.T2StartSize; s <= cfg.T2MaxSize && s <= len(sortedKeys); s *= 2 {
			t2Sizes = append(t2Sizes, s)
		}

		for _, size := range t2Sizes {
			mid := (len(sortedKeys) - size) / 2
			startKey := sortedKeys[mid]
			endKey := sortedKeys[mid+size-1]

			fmt.Printf("[T2] %s: size=%d scanning [%d, %d]...\n", def.Name, size, startKey, endKey)

			start := time.Now()
			it, err := idx.Range(startKey, endKey)
			if err != nil {
				fmt.Printf("[T2] %s: Range() error: %v\n", def.Name, err)
				continue
			}

			keysRead := 0
			for it.Next() {
				keysRead++
			}
			if err := it.Error(); err != nil {
				fmt.Printf("[T2] %s: iterator error: %v\n", def.Name, err)
			}
			it.Close()

			totalDuration := time.Since(start)

			r := T2Result{
				Index:     def.Name,
				RangeSize: size,
				KeysRead:  keysRead,
				TotalMs:   totalDuration.Microseconds(),
				OpsPerSec: float64(keysRead) / totalDuration.Seconds(),
			}

			fmt.Printf("[T2] %s: size=%d keys_read=%d total=%dµs tput=%.0f keys/s\n",
				r.Index, r.RangeSize, r.KeysRead, r.TotalMs, r.OpsPerSec)

			_ = w.Write([]string{
				r.Index,
				strconv.Itoa(r.RangeSize),
				strconv.Itoa(r.KeysRead),
				strconv.FormatInt(r.TotalMs, 10),
				strconv.FormatFloat(r.OpsPerSec, 'f', 2, 64),
			})
		}

		_ = idx.Close()
		if cfg.CleanupData {
			cleanupIndexData(idxPath)
		}
	}

	fmt.Printf("[T2] results written to %s\n", filepath.Join(cfg.OutDir, "t2_range_query.csv"))
	return nil
}

type T3Result struct {
	Index         string
	OpCount       int
	OpsPerSec     float64
	CumulativeOps int
}

var t3Header = []string{
	"index", "op_count", "ops_per_sec", "cumulative_ops",
}

// RunBenchmarkT3 executes the write throughput benchmark (T3).
// It measures how quickly each index can ingest new random key-value pairs.
func RunBenchmarkT3(indices []IndexDef, cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}
	f, err := os.Create(filepath.Join(cfg.OutDir, "t3_write_throughput.csv"))
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write(t3Header)

	rng := rand.New(rand.NewSource(cfg.Seed))

	for _, def := range indices {
		fmt.Printf("[T3] %s: running write throughput...\n", def.Name)

		idxPath := filepath.Join(cfg.DataDir, def.Name+"_t3")
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			continue
		}

		if s, ok := idx.(interface{ SetSyncInterval(int) }); ok {
			s.SetSyncInterval(500)
		}

		windowStart := time.Now()
		windowOps := 0

		for i := 0; i < cfg.WriteOpsTotal; i++ {
			key := rng.Int63()
			val := make([]byte, cfg.ValueSize)
			rng.Read(val)

			if err := idx.Insert(key, val); err != nil {
				break
			}

			windowOps++

			if windowOps >= cfg.WriteOpsWindow {
				duration := time.Since(windowStart).Seconds()
				opsPerSec := float64(windowOps) / duration

				_ = w.Write([]string{
					def.Name,
					strconv.Itoa(i + 1),
					fmt.Sprintf("%.2f", opsPerSec),
					strconv.Itoa(i + 1),
				})

				windowStart = time.Now()
				windowOps = 0
			}

		}
		_ = idx.Close()
		if cfg.CleanupData {
			cleanupIndexData(idxPath)
		}
	}
	return nil
}

type MixedSummaryResult struct {
	Index     string
	OpType    string
	Count     int
	MinNs     int64
	P50Ns     int64
	P95Ns     int64
	P99Ns     int64
	AvgNs     int64
	OpsPerSec float64
}

var mixedSummaryHeader = []string{
	"index", "op_type", "count", "min_ns", "p50_ns", "p95_ns", "p99_ns", "avg_ns", "ops_per_sec",
}

// RunMixedWorkload executes a benchmark with a mix of read and write operations.
func RunMixedWorkload(indices []IndexDef, cfg Config, readPercent int, testLabel string, fileName string) error {
	// Detailed log file
	f, err := os.Create(filepath.Join(cfg.OutDir, fileName))
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"index", "op_count", "responetime_ns", "type"})

	// Summary file
	sumFile, err := os.Create(filepath.Join(cfg.OutDir, fileName[:len(fileName)-len(".csv")]+"_summary.csv"))
	if err != nil {
		return err
	}
	defer sumFile.Close()
	sw := csv.NewWriter(sumFile)
	defer sw.Flush()
	_ = sw.Write(mixedSummaryHeader)

	for _, def := range indices {
		fmt.Printf("[%s] %s: Starting %d/%d workload...\n", testLabel, def.Name, readPercent, 100-readPercent)

		idxPath := filepath.Join(cfg.DataDir, def.Name+"_"+testLabel)
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			continue
		}

		ds := NewDataset(cfg.DatasetSize, cfg.ValueSize, cfg.Seed)
		if err := fillIndex(idx, *ds); err != nil {
			fmt.Printf("[%s] %s: fill failed: %v — skipping\n", testLabel, def.Name, err)
			_ = idx.Close()
			if cfg.CleanupData {
				cleanupIndexData(idxPath)
			}
			continue
		}

		// Re-enable sync for the actual benchmark workload.
		if s, ok := idx.(interface{ SetSyncInterval(int) }); ok {
			s.SetSyncInterval(500)
		}

		rng := rand.New(rand.NewSource(cfg.Seed + 2))

		readTimes := make([]int64, 0, cfg.MixedOpsTotal)
		writeTimes := make([]int64, 0, cfg.MixedOpsTotal)
		startTotal := time.Now()

		for i := 0; i < cfg.MixedOpsTotal; i++ {
			decision := rng.Intn(100)

			if decision < readPercent {
				// READ
				key := ds.Keys[rng.Intn(len(ds.Keys))]
				start := time.Now()
				_, _ = idx.Get(key)
				responetime := time.Since(start).Nanoseconds()
				readTimes = append(readTimes, responetime)

				if i%cfg.LogInterval == 0 {
					_ = w.Write([]string{def.Name, strconv.Itoa(i), strconv.FormatInt(responetime, 10), "read"})
				}
			} else {
				// WRITE
				newKey := rng.Int63()
				val := make([]byte, cfg.ValueSize)
				rng.Read(val)

				start := time.Now()
				_ = idx.Insert(newKey, val)
				responetime := time.Since(start).Nanoseconds()
				writeTimes = append(writeTimes, responetime)

				if i%cfg.LogInterval == 0 {
					_ = w.Write([]string{def.Name, strconv.Itoa(i), strconv.FormatInt(responetime, 10), "write"})
				}
			}
		}
		durationTotal := time.Since(startTotal)
		_ = idx.Close()

		// Calculate and Write Summaries
		processStats := func(times []int64, opType string) {
			if len(times) == 0 {
				return
			}
			sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
			r := MixedSummaryResult{
				Index:     def.Name,
				OpType:    opType,
				Count:     len(times),
				MinNs:     times[0],
				P50Ns:     pct(times, 50),
				P95Ns:     pct(times, 95),
				P99Ns:     pct(times, 99),
				AvgNs:     avg(times),
				OpsPerSec: float64(len(times)) / durationTotal.Seconds(),
			}

			fmt.Printf("[%s] %s %-5s: count=%-6d avg=%-8dns p50=%-8dns p95=%-8dns tput=%-8.0f ops/s\n",
				testLabel, r.Index, r.OpType, r.Count, r.AvgNs, r.P50Ns, r.P95Ns, r.OpsPerSec)

			_ = sw.Write([]string{
				r.Index, r.OpType, strconv.Itoa(r.Count),
				strconv.FormatInt(r.MinNs, 10), strconv.FormatInt(r.P50Ns, 10),
				strconv.FormatInt(r.P95Ns, 10), strconv.FormatInt(r.P99Ns, 10),
				strconv.FormatInt(r.AvgNs, 10), strconv.FormatFloat(r.OpsPerSec, 'f', 2, 64),
			})
		}

		processStats(readTimes, "read")
		processStats(writeTimes, "write")

		if cfg.CleanupData {
			cleanupIndexData(idxPath)
		}
	}
	return nil
}

// RunBenchmarkT4 executes the read-heavy mixed workload benchmark (T4).
// It uses a 95% read / 5% write ratio.
func RunBenchmarkT4(indices []IndexDef, cfg Config) error {
	return RunMixedWorkload(indices, cfg, 95, "T4", "t4_read_heavy.csv")
}

// RunBenchmarkT5 executes the write-heavy mixed workload benchmark (T5).
// It uses a 5% read / 95% write ratio.
func RunBenchmarkT5(indices []IndexDef, cfg Config) error {
	return RunMixedWorkload(indices, cfg, 5, "T5", "t5_write_heavy.csv")
}

//---

func avg(responetimes []int64) int64 {
	var sum int64
	for _, v := range responetimes {
		sum += v
	}
	return sum / int64(len(responetimes))
}

func pct(responetimes []int64, p int) int64 {
	i := (p * len(responetimes)) / 100
	if i >= len(responetimes) {
		i = len(responetimes) - 1
	}
	return responetimes[i]
}
