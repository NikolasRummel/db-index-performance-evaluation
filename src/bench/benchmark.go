package bench

import (
	"fmt"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/bptree"
	"github.com/btree-query-bench/bmark/dbms/index/btree"
	"github.com/btree-query-bench/bmark/dbms/index/lsm"
)

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
}

type IndexDef struct {
	Name    string
	NewFunc func(path string) (index.Index, error)
}

func Indexes(cfg Config) []IndexDef {
	return []IndexDef{
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
				return lsm.Open(path)
			},
		},
	}
}

func RunBenchmarks(cfg Config) error {
	indices := Indexes(cfg)
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("create results dir: %w", err)
	}
	if err := RunBenchmarkT1(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT2(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT3(indices, cfg); err != nil {
		return err
	}
	if err := RunBenchmarkT4(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT5(indices, cfg); err != nil {
		return err
	}

	return nil
}
