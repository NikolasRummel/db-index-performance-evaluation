package main

import (
	"log"

	"github.com/btree-query-bench/bmark/bench"
)

func main() {
	log.Printf("Welcome to BTree Query Benchmark!\n\n")
	cfg := bench.Config{
		Seed:            42,
		CachePages:      512,
		DataDir:         "./out/data",
		OutDir:          "./out/results",
		DatasetSize:     100_000,
		PointQueryCount: 5_000,
		WriteOpsTotal:   50_000,
		WriteOpsWindow:  5_000,
		MixedOpsTotal:   20_000,
		LogInterval:     500,
		ValueSize:       64,
	}
	if err := bench.RunBenchmarks(cfg); err != nil {
		log.Fatalf("benchmark failed: %v", err)
	}
	if err := bench.PlotAll(cfg.OutDir); err != nil {
		log.Fatalf("plot failed: %v", err)
	}

}
