# Analysis and Comparison of Database Index Structures

This repository contains the source code for a student research project (Studienarbeit) at DHBW Karlsruhe, focusing on the analysis and comparison of various database index structures.

The project implements and benchmarks three primary indexing strategies:
- **B-Tree**: A classic disk-based balanced tree.
- **B+ Tree**: An optimization of the B-Tree, storing all data in leaf nodes for better range scans.
- **LSM-Tree (Log-Structured Merge-Tree)**: Optimized for write-heavy workloads (implemented using [Pebble](https://github.com/cockroachdb/pebble)).

## Project Structure

- `src/`: Go implementation of the indexing structures and the benchmarking suite.
  - `dbms/index/`: Implementation of B-Tree, B+ Tree, and LSM-Tree wrappers.
  - `bench/`: Benchmarking logic, dataset generation, and plotting.
- `thesis/`: The written thesis in [Typst](https://typst.app/).
  - `sections/`: Individual chapters of the thesis.
  - `assets/`: Images and result plots used in the document.
- `out/`: Benchmark results.
  - `data/`: Temporary data files generated during benchmarks.
  - `results/`: CSV files, HTML reports, and generated plots.

## Benchmarks

The benchmark suite evaluates the indices across several dimensions:
1. **T1: Point Query**: Latency and throughput of single-key lookups.
2. **T2: Range Query**: Performance of retrieving ranges of various sizes.
3. **T3: Write Throughput**: Ingestion speed for large datasets.
4. **T4: Read-Heavy Workload**: Mixed operations with 90% reads.
5. **T5: Write-Heavy Workload**: Mixed operations with 90% writes.

## Getting Started

### Prerequisites

- **Go**: Version 1.24 or higher.
- **Typst**: To compile the thesis document.

### Running Benchmarks

To execute the full benchmarking suite:

```bash
cd src
go run main.go
```

You can customize the benchmark parameters using flags:

```bash
go run main.go --dataset-size 1000000 --point-queries 500000
```

Run `go run main.go --help` to see all available options.

### Viewing Results

After running the benchmarks, the results are stored in `out/results/`.
- **CSV files**: Raw data for each test.
- **HTML files**: Interactive charts generated using `go-echarts`. Open these in your browser to visualize the results.

### Building the Thesis

To compile the PDF of the thesis:

```bash
cd thesis
typst compile main.typ
```

## License

This project is for educational purposes as part of a student research project.
