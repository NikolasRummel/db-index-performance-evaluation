package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/btree-query-bench/bmark/benchmark"
	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/bptree"
	"github.com/btree-query-bench/bmark/dbms/index/btree"
	"github.com/btree-query-bench/bmark/dbms/index/lsm"
)

const outputRoot = "results"

var (
	baseDir    = filepath.Join(outputRoot, "data")
	resultsCSV = filepath.Join(outputRoot, "results.csv")
	plotsDir   = filepath.Join(outputRoot, "plots")
)

func main() {
	if err := os.MkdirAll(outputRoot, 0755); err != nil {
		log.Fatalf("failed to create output root: %v", err)
	}

	if !resultsExist() {
		fmt.Println("No results found...")
		if err := runBenchmark(); err != nil {
			log.Fatalf("benchmark: %v", err)
		}
	}

	fmt.Printf("Generating plots in %s...\n", plotsDir)
	if err := benchmark.GeneratePlots(resultsCSV, plotsDir); err != nil {
		log.Fatalf("plots: %v", err)
	}

	fmt.Printf("\nFinished!")
}

func resultsExist() bool {
	_, err := os.Stat(resultsCSV)
	return err == nil
}

func runBenchmark() error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	cfg := benchmark.DefaultConfig()

	if v := os.Getenv("BENCH_OPS"); v != "" {
		var n int
		fmt.Sscan(v, &n)
		cfg.Ops = n
		cfg.PreloadSize = n / 2
		cfg.TotalWriteOps = n * 3
		cfg.WriteWindowSize = max(n/20, 100)
		cfg.DatasetSizes = []int{n / 10, n / 4, n / 2, n, n * 2}
		fmt.Printf("Quick mode enabled: Ops=%d\n\n", n)
	}

	engines := []benchmark.EngineFactory{
		{
			Name: "btree",
			NewFunc: func(path string) (index.Index, error) {
				return btree.Open(path, cfg.CachePages)
			},
		},
		{
			Name: "bptree",
			NewFunc: func(path string) (index.Index, error) {
				return bptree.Open(path, cfg.CachePages)
			},
		},
		{
			Name: "lsm_pebble",
			NewFunc: func(path string) (index.Index, error) {
				if err := os.MkdirAll(path, 0755); err != nil {
					return nil, err
				}
				return lsm.Open(path)
			},
		},
	}

	runner := benchmark.NewRunner(cfg, engines, baseDir)

	return runner.RunAll()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
