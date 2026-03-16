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
	Index               string
	NDataset            int
	NQueries            int
	MinNs               int64
	Q1Ns                int64
	P50Ns               int64
	Q3Ns                int64
	MaxNs               int64
	AvgNs               int64
	P95Ns               int64
	P99Ns               int64
	ThroughputOpsPerSec float64
	TotalMs             int64
}

var t1Header = []string{
	"index", "n_dataset", "n_queries",
	"min_ns", "q1_ns", "p50_ns", "q3_ns", "max_ns",
	"avg_ns", "p95_ns", "p99_ns",
	"throughput_ops_sec", "total_ms",
}

func fillIndex(idx index.Index, ds Dataset) error {
	for i, k := range ds.Keys {
		if err := idx.Insert(k, ds.Values[i]); err != nil {
			return fmt.Errorf("insert key %d: %w", k, err)
		}
	}
	return nil
}

func RunBenchmarkT1(indices []IndexDef, cfg Config) error {
	ds := NewDataset(cfg.T1N, cfg.Seed)
	queryKeys := ds.RandomKeys(cfg.T1NQuery)

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
		fmt.Printf("[T1] %s: filling index with %d keys...\n", def.Name, cfg.T1N)

		idxPath := filepath.Join(cfg.DataDir, def.Name+"_t1")
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			fmt.Printf("[T1] %s: open failed: %v — skipping\n", def.Name, err)
			continue
		}

		if err := fillIndex(idx, *ds); err != nil {
			fmt.Printf("[T1] %s: fill failed: %v — skipping\n", def.Name, err)
			_ = idx.Close()
			continue
		}

		if h, ok := idx.(interface{ Height() int }); ok {
			fmt.Printf("[T1] %s: tree height = %d\n", def.Name, h.Height())
		}
		if l, ok := idx.(interface{ Levels() string }); ok {
			fmt.Printf("[T1] %s: lsm levels = %s\n", def.Name, l.Levels())
		}

		fmt.Printf("[T1] %s: running %d point queries...\n", def.Name, cfg.T1NQuery)

		lats := make([]int64, 0, cfg.T1NQuery)
		start := time.Now()

		for _, key := range queryKeys {
			t := time.Now()
			val, e := idx.Get(key)
			lats = append(lats, time.Since(t).Nanoseconds())
			if e != nil {
				fmt.Printf("[T1] %s: Get(%d) error: %v\n", def.Name, key, e)
			} else if val == nil {
				fmt.Printf("[T1] %s: key %d not found\n", def.Name, key)
			}
		}

		totalDuration := time.Since(start)
		_ = idx.Close()

		sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

		r := T1Result{
			Index:               def.Name,
			NDataset:            cfg.T1N,
			NQueries:            cfg.T1NQuery,
			MinNs:               lats[0],
			Q1Ns:                pct(lats, 25),
			P50Ns:               pct(lats, 50),
			Q3Ns:                pct(lats, 75),
			MaxNs:               lats[len(lats)-1],
			AvgNs:               avg(lats),
			P95Ns:               pct(lats, 95),
			P99Ns:               pct(lats, 99),
			ThroughputOpsPerSec: float64(cfg.T1NQuery) / totalDuration.Seconds(),
			TotalMs:             totalDuration.Milliseconds(),
		}

		fmt.Printf("[T1] %s: min=%dns q1=%dns p50=%dns q3=%dns max=%dns avg=%dns p95=%dns p99=%dns tput=%.0f ops/s\n",
			r.Index, r.MinNs, r.Q1Ns, r.P50Ns, r.Q3Ns, r.MaxNs, r.AvgNs, r.P95Ns, r.P99Ns, r.ThroughputOpsPerSec)

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
			strconv.FormatFloat(r.ThroughputOpsPerSec, 'f', 2, 64),
			strconv.FormatInt(r.TotalMs, 10),
		})
	}

	fmt.Printf("[T1] results written to %s\n", filepath.Join(cfg.OutDir, "t1_point_query.csv"))
	return nil
}

type T2Result struct {
	Index                string
	RangeSize            int
	KeysRead             int
	TotalMs              int64
	ThroughputKeysPerSec float64
}

var t2Header = []string{
	"index", "range_size", "keys_read",
	"total_ms", "throughput_keys_per_sec",
}

func RunBenchmarkT2(indices []IndexDef, cfg Config) error {
	ds := NewDataset(cfg.T1N, cfg.Seed)
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
		fmt.Printf("[T2] %s: filling index with %d keys...\n", def.Name, cfg.T1N)

		idxPath := filepath.Join(cfg.DataDir, def.Name+"_t2")
		idx, err := def.NewFunc(idxPath)
		if err != nil {
			fmt.Printf("[T2] %s: open failed: %v — skipping\n", def.Name, err)
			continue
		}

		if err := fillIndex(idx, *ds); err != nil {
			fmt.Printf("[T2] %s: fill failed: %v — skipping\n", def.Name, err)
			_ = idx.Close()
			continue
		}

		var t2Sizes []int
		for s := 4096; s <= 500_000; s *= 2 {
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
				Index:                def.Name,
				RangeSize:            size,
				KeysRead:             keysRead,
				TotalMs:              totalDuration.Microseconds(),
				ThroughputKeysPerSec: float64(keysRead) / totalDuration.Seconds(),
			}

			fmt.Printf("[T2] %s: size=%d keys_read=%d total=%dµs tput=%.0f keys/s\n",
				r.Index, r.RangeSize, r.KeysRead, r.TotalMs, r.ThroughputKeysPerSec)

			_ = w.Write([]string{
				r.Index,
				strconv.Itoa(r.RangeSize),
				strconv.Itoa(r.KeysRead),
				strconv.FormatInt(r.TotalMs, 10),
				strconv.FormatFloat(r.ThroughputKeysPerSec, 'f', 2, 64),
			})
		}

		_ = idx.Close()
	}

	fmt.Printf("[T2] results written to %s\n", filepath.Join(cfg.OutDir, "t2_range_query.csv"))
	return nil
}

type T3Result struct {
	Index             string
	TimeSec           int
	InsertsThisSec    int
	CumulativeInserts int
}

var t3Header = []string{
	"index", "time_sec", "inserts_this_sec", "cumulative_inserts",
}

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

		windowStart := time.Now()
		windowOps := 0

		for i := 0; i < cfg.TotalWriteOps; i++ {
			key := rng.Int63()
			val := make([]byte, ValueSize)
			rng.Read(val)

			if err := idx.Insert(key, val); err != nil {
				break
			}

			windowOps++

			if windowOps >= cfg.WriteWindowSize {
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
	}
	return nil
}

func avg(lats []int64) int64 {
	var sum int64
	for _, v := range lats {
		sum += v
	}
	return sum / int64(len(lats))
}

func pct(lats []int64, p int) int64 {
	i := (p * len(lats)) / 100
	if i >= len(lats) {
		i = len(lats) - 1
	}
	return lats[i]
}
