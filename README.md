# Database Index Comparison

> Performance evaluation of B-Tree, B+-Tree, LSM-Tree, and Sequential List index structures in primary memory.

**Studienarbeit** - DHBW Karlsruhe - 2025

## Description

This project implements and benchmarks four different index structures to analyze their performance characteristics under various workload patterns (OLTP/OLAP scenarios). The implementation is done in Go with visualization in Julia.

## Structure
```
├── src/           # Go implementation of index structures
├── benchmarks/    # Benchmark definitions (JSON)
├── results/       # Benchmark results and plots
├── thesis/        # Typst thesis document
└── scripts/       # Helper scripts (run benchmarks, visualize)
```

## Index Structures

- **B-Tree** - Classic balanced search tree
- **B+-Tree** - Optimized for range queries  
- **LSM-Tree** - Write-optimized log-structured merge tree
- **Sequential List** - Baseline comparison
