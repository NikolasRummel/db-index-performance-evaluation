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
	Cache           int
	T1N             int
	T1NQuery        int
	TotalWriteOps   int
	WriteWindowSize int
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
				return btree.Open(path, cfg.Cache)
			},
		},
		{
			Name: "bptree",
			NewFunc: func(path string) (index.Index, error) {
				return bptree.Open(path, cfg.Cache)
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
	if err := RunBenchmarkT1(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT2(indices, cfg); err != nil {
		return err
	}

	if err := RunBenchmarkT3(indices, cfg); err != nil {
		return err
	}

	return nil
}
