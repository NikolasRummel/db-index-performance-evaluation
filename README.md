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

The benchmark suite evaluates the indices across several dimensions and configurations:
1. **T1: Point Query**: Latency and throughput of single-key lookups.
2. **T2: Range Query**: Performance of retrieving ranges of various sizes.
3. **T3: Write Throughput**: Ingestion speed for large datasets.
4. **T4: Read-Heavy Workload**: Mixed operations with 90% reads.
5. **T5: Write-Heavy Workload**: Mixed operations with 90% writes.

### Index Variants

To provide a deeper analysis, the suite compares several implementation variants:
- **B-Tree & B+ Tree**: Tested with different page sizes (**4KB, 8KB, 16KB**) to analyze the impact on I/O.
- **LSM-Tree**: Tested with varying memtable sizes (**16MB, 32MB, 64MB**) using the Pebble engine.

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

#### Configuration Flags

The suite is highly configurable for different hardware and test scenarios:

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--seed` | `42` | Seed for reproducibility of random data. |
| `--dataset-size` | `5,000,000` | Number of entries in the initial dataset. |
| `--cache-pages` | `4096` | Number of pages kept in the internal buffer cache. |
| `--value-size` | `128` | Size of each value in bytes. |
| `--cleanup-data` | `true` | Delete large temporary DB files after each test run. |

Run `go run main.go --help` to see the full list of parameters.

### Viewing Results

After running the benchmarks, results are stored in `out/results/`:
- **CSV files**: Raw data for further analysis.
- **HTML files**: **Interactive charts** generated via `go-echarts`. These allow zooming, filtering by index type, and detailed inspection of data points.

### Building the Thesis

To compile the PDF of the thesis:

```bash
cd thesis
typst compile main.typ
```

## License

This project is for educational purposes as part of a student research project.
