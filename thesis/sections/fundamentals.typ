
#pagebreak()

= Fundamentals

== Index Structures in Database Systems


== Search Tree-Based Index Structures
#let p = $p$
In order to understand the most common data structure for DBMS, the B+-Tree @elmasri2016 [p. 618], we will start with standard search trees.

A search tree is a type of data structure used to organize and manage data in a way that allows for efficient searching. It is defined with an order #p so that each node contains at most $p - 1$ keys and $p$ pointers in the sequence $<P_1, K_1, P_2, K_2, ..., P_(q-1), K_(q-1), P_q>$, where $q <= p$. Each $K_i$ is a key which we are searching for in our tree data structure, whereas each $P_i$ represents the pointers to child nodes (including NULL pointers) @elmasri2016 [p. 618]. 

In order to be a search tree, this tree must satisfy the following constraints:

1. In each node, the $K_i$ values are ordered such that $K_1 < K_2 < dots < K_(p-1)$ holds.
2. For all values $X$ in the subtree rooted at node $P_i$, the following conditions hold:
  - For $1 < i < q$: $K_(i-1) < X < K_i$
  - For $i = 1$: $X < K_1$
  - For $i = q$: $K_(q-1) < X$

A search tree can then be visualized as follows:


By utilizing an index, a search for a specific key can be performed. For example, if we search for the key $12$, we follow the appropriate child pointers based on the key comparisons at each node. 

To determine the time complexity, the efficiency depends heavily on how the tree grows over time. It is clear that for deeper trees, a search operation will require more time. In the worst-case scenario for a unbalenced search tree, we encounter a time complexity of $O(n)$, because a normal search tree cannot guarantee a balanced structure.

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