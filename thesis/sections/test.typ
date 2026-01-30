

#pagebreak()

= Design of the Comparison <design>

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

= Evaluation and Analysis <evaluation>

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

= Conclusion <conclusion>

== Summary of Findings

== Achieved Goals

== Limitations

#pagebreak()

= Outlook <outlook>

== Potential Extensions

== Future Research Directions

#pagebreak()

= References

#pagebreak()

= Appendix

== Complete Benchmark Results

== JSON Schema Specifications

== Additional Figures and Tables