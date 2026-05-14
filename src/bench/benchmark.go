// Package bench provides benchmarking tools and test cases for comparing
// different database index implementations.
package bench

import (
	"fmt"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/bptree"
	"github.com/btree-query-bench/bmark/dbms/index/btree"
	"github.com/btree-query-bench/bmark/dbms/index/lsm"
)

// Config defines the configuration parameters for the benchmark suite.
type Config struct {
	Seed            int64
	OutDir          string
	DataDir         string
	CachePages      int
	DatasetSize     int
	PointQueryCount int
	WriteOpsTotal   int
	WriteOpsWindow  int
	MixedOpsTotal   int
	LogInterval     int
	ValueSize       int
	T2StartSize     int
	T2MaxSize       int
	CleanupData     bool
}

// IndexDef defines an index implementation and a factory function to create it.
type IndexDef struct {
	Name    string
	NewFunc func(path string) (index.Index, error)
}

// Indexes returns a slice of index implementations to be benchmarked.
func Indexes(cfg Config) []IndexDef {
	return []IndexDef{
		{
			Name: "btree_4k",
			NewFunc: func(path string) (index.Index, error) {
				return btree.Open(path, cfg.CachePages, 4096)
			},
		},
		{
			Name: "btree_8k",
			NewFunc: func(path string) (index.Index, error) {
				return btree.Open(path, cfg.CachePages, 8192)
			},
		},
		{
			Name: "btree_16k",
			NewFunc: func(path string) (index.Index, error) {
				return btree.Open(path, cfg.CachePages, 16384)
			},
		},
		{
			Name: "bptree_4k",
			NewFunc: func(path string) (index.Index, error) {
				return bptree.Open(path, cfg.CachePages, 4096)
			},
		},
		{
			Name: "bptree_8k",
			NewFunc: func(path string) (index.Index, error) {
				return bptree.Open(path, cfg.CachePages, 8192)
			},
		},
		{
			Name: "bptree_16k",
			NewFunc: func(path string) (index.Index, error) {
				return bptree.Open(path, cfg.CachePages, 16384)
			},
		},
		{
			Name: "lsm_pebble_16m",
			NewFunc: func(path string) (index.Index, error) {
				return lsm.Open(path, 16)
			},
		},
		{
			Name: "lsm_pebble_32m",
			NewFunc: func(path string) (index.Index, error) {
				return lsm.Open(path, 32)
			},
		},
		{
			Name: "lsm_pebble_64m",
			NewFunc: func(path string) (index.Index, error) {
				return lsm.Open(path, 64)
			},
		},
	}
}

// RunBenchmarks runs the full suite of benchmarks for all index implementations defined in the configuration.
func RunBenchmarks(cfg Config) error {
	indices := Indexes(cfg)
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("create results dir: %w", err)
	}
	/*if err := RunBenchmarkT1(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT2(indices, cfg); err != nil {
		return err
	}
		if err := RunBenchmarkT3(indices, cfg); err != nil {
		return err
	}
	*/
	if err := RunBenchmarkT4(indices, cfg); err != nil {
		return err
	}
	if err := RunBenchmarkT5(indices, cfg); err != nil {
		return err
	}

	return nil
}
