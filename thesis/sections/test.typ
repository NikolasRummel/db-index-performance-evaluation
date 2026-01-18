= Introduction

== Motivation

== Problem Statement and Relevance

== Objectives and Research Questions

== Structure of the Thesis

#pagebreak()

= Fundamentals

== Index Structures in Database Systems

=== Definition and Purpose

=== Requirements for Index Structures

=== Classification of Index Structures

== Search Tree-Based Index Structures

=== B-Trees
==== Structure and Properties
==== Operations
===== Search
===== Insert
===== Delete
==== Complexity Analysis

=== B+-Trees
==== Differences from B-Trees
==== Advantages for Range Queries
==== Implementation Aspects

== LSM-Trees (Log-Structured Merge Trees)

=== Structure and Concept

=== Write-Optimized Design

=== MemTable and SSTables

=== Compaction Process

=== Read Path and Bloom Filters

=== Use Cases

== Workload Characteristics

=== OLTP Scenarios

=== OLAP Scenarios

== Index Structures in Real Database Systems

=== PostgreSQL
// B+-Trees

=== MySQL/InnoDB
// B+-Trees with clustered indexes

=== SQLite
// B+-Trees

=== RocksDB / LevelDB
// LSM-Trees

=== Cassandra / ScyllaDB
// LSM-Trees

=== When Databases Use No Index
// Full table scans, small tables

#pagebreak()

= Design of the Comparison

== Overview of the Implementation

=== Goals and Scope

=== Component Diagram
// Component diagram showing:
// - IndexInterface (central interface)
// - Index Implementations: B-Tree, B+-Tree, LSM-Tree, Sequential List
// - Benchmark Runner
// - JSON Parser (benchmark.json → operations)
// - Statistics Collector
// - Results Writer (→ results.json)
// - Data flow between components

=== Common Interface Design
// IndexInterface that all structures implement

== Selected Index Structures

=== B-Tree

=== B+-Tree

=== LSM-Tree

=== No Index (Sequential List/Array)
// Baseline comparison

== Implementation Details

=== Technology Stack: Go (Golang)

=== Data Structures

=== Algorithms

=== Optimizations

== Development of Each Index

=== B-Tree Implementation
==== Node Structure
// Code included in text
==== Insert Operation
// Code included in text
==== Search Operation
// Code included in text
==== Delete Operation
// Code included in text
==== Range Query
// Code included in text

=== B+-Tree Implementation
==== Node Structure
// Code included in text
==== Leaf Node Chaining
// Code included in text
==== Insert Operation
// Code included in text
==== Search Operation
// Code included in text
==== Delete Operation
// Code included in text
==== Range Query Optimization
// Code included in text

=== LSM-Tree Implementation
==== MemTable Structure
// Code included in text
==== SSTable Format
// Code included in text
==== Write Path
// Code included in text
==== Compaction Strategy
// Code included in text
==== Read Path with Bloom Filters
// Code included in text

=== No Index Implementation
==== Simple Array/List Structure
// Code included in text
==== Sequential Operations
// Code included in text

#pagebreak()

= Benchmark Framework

== Benchmark Design

=== Requirements

=== Design Decisions

=== Metrics

== Data Generation

=== Sequential Data

=== Random Data

=== Skewed Distributions

=== Real-World Patterns

== JSON-Based Interface

=== Benchmark Specification Format

=== Results Format

=== Examples

== Implementation

=== Benchmark Runner
// Code will be included here

=== JSON Parser
// Code will be included here

=== Operation Execution
// Code will be included here

=== Timing and Statistics Collection
// Code will be included here

=== Testing and Validation
// Code examples and test results included

#pagebreak()

= Evaluation

== Benchmark Scenarios

=== Sequential Insertions

=== Random Insertions

=== Point Queries

=== Range Queries

=== Write-Heavy Workloads
==== High Insert Rate
==== LSM-Tree Advantage
==== B-Tree Write Amplification

=== Mixed Workloads
==== OLTP-like Workload
==== OLAP-like Workload

=== Read-Heavy Workloads
==== B+-Tree Advantage
==== LSM-Tree Read Amplification

=== Delete Operations

== Test Environment

=== Hardware Specifications

=== Software Configuration

=== Benchmark Parameters

== Results

=== Insert Performance
==== Sequential vs Random
==== B-Tree vs B+-Tree vs LSM-Tree vs No Index
==== Impact of Degree Parameter
==== Write Amplification Analysis
==== Scalability

=== Search Performance
==== Point Queries
==== B-Tree vs B+-Tree vs LSM-Tree vs No Index
==== Impact of Tree Height
==== Bloom Filter Effectiveness (LSM)
==== Cache Effects

=== Range Query Performance
==== B-Tree vs B+-Tree vs LSM-Tree vs No Index
==== Scaling with Range Size
==== Sequential Scan Advantage

=== Memory Consumption
==== Memory per Node/Structure
==== Total Memory Footprint
==== LSM-Tree Memory Overhead (MemTable + SSTables)
==== Trade-offs

=== Write-Heavy Workload Results
==== LSM-Tree Performance
==== B-Tree Performance
==== Compaction Impact

=== Read-Heavy Workload Results
==== B+-Tree Performance
==== LSM-Tree Read Amplification
==== No Index Baseline

=== Mixed Workload Results
==== OLTP Scenario
==== OLAP Scenario
==== Best Structure per Scenario

== Analysis

=== Interpretation of Results

=== Strengths and Weaknesses of Each Structure
==== B-Tree
==== B+-Tree
==== LSM-Tree
==== No Index

=== Write vs Read Trade-offs

=== Application Recommendations
==== When to Use B-Trees
==== When to Use B+-Trees
==== When to Use LSM-Trees
==== When No Index is Sufficient

=== Comparison with Real Database Systems
==== Confirmed Properties
==== Differences in Practice
==== Why RocksDB Uses LSM
==== Why PostgreSQL Uses B+-Trees
==== Lessons Learned

#pagebreak()

= Conclusion

== Summary of Findings

== Achieved Goals

== Limitations

#pagebreak()

= Outlook

== Potential Extensions

== Future Research Directions

#pagebreak()

= References

#pagebreak()

= Appendix

== Complete Benchmark Results
// Tables with detailed measurements

== JSON Schema Specifications
// Schema definitions

== Additional Figures and Tables
// Supplementary visualizations