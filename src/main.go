package main

import (
	"log"

	"github.com/btree-query-bench/bmark/bench"
)

func main() {
	log.Printf("Welcome to BTree Query Benchmark!\n\n")
	cfg := bench.Config{
		Seed:            42,
		Cache:           512,
		DataDir:         "./out/data",
		OutDir:          "./out/results",
		T1N:             1_000_000,
		T1NQuery:        10_000,
		TotalWriteOps:   500_000,
		WriteWindowSize: 10_000,
		TotalMixedOps:   200000,
		LogInterval:     100,
	}
	if err := bench.RunBenchmarks(cfg); err != nil {
		log.Fatalf("benchmark failed: %v", err)
	}
	if err := bench.PlotAll(cfg.OutDir); err != nil {
		log.Fatalf("plot failed: %v", err)
	}

}
