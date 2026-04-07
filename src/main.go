package main

import (
	"flag"
	"log"

	"github.com/btree-query-bench/bmark/bench"
)

// ReadFlagsOrDefault parses command-line flags or returns a Config with default values.
func ReadFlagsOrDefault() bench.Config {
	cfg := bench.Config{}

	flag.Int64Var(&cfg.Seed, "seed", 42, "Random seed")
	flag.IntVar(&cfg.CachePages, "cache-pages", 512, "Number of cache pages")
	flag.StringVar(&cfg.DataDir, "data-dir", "./out/data", "Directory for data files")
	flag.StringVar(&cfg.OutDir, "out-dir", "./out/results", "Directory for result output")
	flag.IntVar(&cfg.DatasetSize, "dataset-size", 1_000_000, "Number of entries in the dataset")
	flag.IntVar(&cfg.PointQueryCount, "point-queries", 500_000, "Number of point queries to run")
	flag.IntVar(&cfg.WriteOpsTotal, "write-ops-total", 1_000_000, "Total write operations")
	flag.IntVar(&cfg.WriteOpsWindow, "write-ops-window", 50_000, "Write ops window size")
	flag.IntVar(&cfg.MixedOpsTotal, "mixed-ops-total", 1_000_000, "Total mixed operations")
	flag.IntVar(&cfg.LogInterval, "log-interval", 500, "Logging interval")
	flag.IntVar(&cfg.ValueSize, "value-size", 64, "Size of each value in bytes")
	flag.IntVar(&cfg.T2StartSize, "t2-start-size", 4096, "T2 range query start size")
	flag.IntVar(&cfg.T2MaxSize, "t2-max-size", 2_000_000, "T2 range query max size")
	flag.Parse()

	return cfg
}

func main() {
	//input
	cfg := ReadFlagsOrDefault()

	// process
	if err := bench.RunBenchmarks(cfg); err != nil {
		log.Fatalf("benchmark failed: %v", err)
	}

	// output
	if err := bench.PlotAll(cfg.OutDir); err != nil {
		log.Fatalf("plot failed: %v", err)
	}
}
