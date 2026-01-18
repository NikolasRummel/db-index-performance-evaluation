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

=== MySQL/InnoDB

=== SQLite

=== RocksDB / LevelDB

=== Cassandra / ScyllaDB

=== When Databases Use No Index

#pagebreak()

= Design of the Comparison

== Overview of the Implementation

=== Goals and Scope

=== Component Diagram

=== Common Interface Design

== Selected Index Structures

== Implementation Details

=== Technology Stack: Go (Golang)

== Development of Each Index

=== B-Tree Implementation
==== Node Structure
==== Insert Operation
==== Search Operation
==== Delete Operation
==== Range Query

=== B+-Tree Implementation
==== Node Structure
==== Leaf Node Chaining
==== Insert Operation
==== Search Operation
==== Delete Operation
==== Range Query Optimization

=== LSM-Tree Implementation
==== MemTable Structure
==== SSTable Format
==== Write Path
==== Compaction Strategy
==== Read Path with Bloom Filters

=== No Index Implementation
==== Simple Array/List Structure
==== Sequential Operations

== Development of the Benchmark 

#pagebreak()

= Evaluation and Analysis

== Experimental Setup

=== Hardware Specifications

=== Software Configuration

=== Benchmark Parameters and Data Generation

== Performance Analysis by Operation

=== Insertion Performance
==== Sequential vs. Random Results
==== Discussion: Write Amplification and Page Splits
==== Impact of the Degree Parameter ($k$)

=== Search and Point Query Performance
==== Latency Comparison: B-Tree vs. B+-Tree vs. LSM
==== Discussion: The Cost of Read Amplification in LSM-Trees
==== Effectiveness of Bloom Filters

=== Range Query Performance
==== B+-Tree Scanning vs. LSM Merging
==== Discussion: Sequential Access Patterns and Iterator Overhead

=== Memory and Storage Footprint
==== Memory Consumption per Structure
==== Disk Space Efficiency and Compaction Impact

== Scenario-Based Synthesis

=== Write-Heavy Workloads
==== Analysis: Why LSM-Trees Dominate High-Ingestion Scenarios

=== Read-Heavy Workloads
==== Analysis: B+-Tree Consistency and Cache Efficiency

=== Mixed Workloads (OLTP vs. OLAP)
==== Evaluation of Tail Latencies and Throughput Stability

== Summary of Strengths and Weaknesses

=== Comparative Matrix of Index Structures

=== Application Recommendations

=== Reflection on Real Database Systems
==== Validating PostgreSQL (B+-Tree) and RocksDB (LSM) Design Choices

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

== JSON Schema Specifications

== Additional Figures and Tables