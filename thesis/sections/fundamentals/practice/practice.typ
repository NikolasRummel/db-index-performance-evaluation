=== Index Structures in practice

==== PostgreSQL
'PostgreSQL' is ....
PostgreSQL uses a special Btree proposed by Lehman, Philip L. and Yao, s. Bing @postgres_btree_paper, which optimizes the performance of concurrent transactions. The B-tree is organized as a B+-tree, where the leaf nodes contain the actual data records, and the internal nodes contain pointers to the leaf nodes.

==== SQLite
'SQLite' was initially released in 2005 and has since become the most widely deployed database engine in the world.  @sqlite_most_deployed It is written in 'C' and was designed to be integrated in embedded systems like mobile applications, therefore being very lightweight @sqlite_general and easy to use. 
Based on SQLite's documentation @sqlite_fileformat, it stores data in a single file and uses two types of B-trees: one for tables and another for indexes. The B-tree for tables is used to store the actual data, while the B-tree for indexes is used to store the index entries.
The B-tree for tables is organized as a B+-tree, where the leaf nodes contain the actual data records, and the internal nodes contain pointers to the leaf nodes. 
==== MongoDB

==== Cassandra
==== CockroachDB
==== ...

TODO: Section about Use cases OLTP, OLAP, WRITE HEAVY, READ HEAVY!
