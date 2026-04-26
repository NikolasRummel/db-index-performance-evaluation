
#import "@preview/clean-dhbw:0.4.0": *

== Practical Relevance and Industrial Application <practice>

While choosing an index structure to design a #gls("DBMS"), the choice between B+ Trees and LSM-Trees is primarily dictated by the expected ratio of read-to-write operations and the target storage medium. While B+ Trees remain the foundation for traditional transactional systems, LSM-Trees have emerged as a dominant architecture for NoSQL databases and key-value stores @lsm_survey[p. 1]. 

=== B+ Tree-Based Systems <bplus_practice>
Traditional relational #gls("DBMS") like PostgreSQL and MySQL primarily utilize B+ Tree index structures to maintain ACID compliance and optimize for OLTP workloads @mysql_innodb. 

`MySQL` was first released in 1995 and has become a widely used #gls("DBMS") with over 1 billion installations @mysql_release. The default storage engine for MySQL, `InnoDB`, data is organized into B-trees @mysql_physical_structure. Since the data is stored in the leaf nodes, `InnoDB`'s B-tree is technically a B+-Tree. To optimize performance, `InnoDB` uses 16KB pages by default. For sequential inserts, it keeps pages 15/16 full to allow for future growth. For random inserts, the fill factor varies between 50% and 93.75%. If a page's data drops below the 50%  due to deletions, `InnoDB` automatically merges neighboring pages to prevent fragmentation and optimize the on-disk footprint @mysql_physical_structure.

`PostgreSQL` is a open soruce relational #gls("DBMS") that uses a special B-Tree proposed by Lehman, Philip L. and Yao, s. Bing @postgres_btree_paper, which optimizes the performance of concurrent transactions. Like in Mysql, the B-tree is organized as a B+-tree, where the leaf nodes contain the actual data records, and the internal nodes contain pointers to the leaf nodes. 

`SQLite` was initially released in 2005 and has since become the most widely deployed database engine in the world @sqlite_most_deployed. It is written in `C` and was designed to be integrated in embedded systems like mobile applications, therefore being very lightweight @sqlite_general and easy to use. 
Based on SQLite's documentation @sqlite_fileformat, it stores data in a single file and uses two types of B-trees: one for tables and another for indexes. The B-tree for tables is used to store the actual data, while the B-tree for indexes is used to store the index entries.
The B-tree for tables is organized as a B+-tree, where again the leaf nodes contain the actual data records, while the internal nodes only contain pointers to the leaf nodes. 

=== LSM-Tree-Based Systems
There are different applications of LSM-Trees, ranging from key-value stores to time-series databases and other systems. 

`Google Bigtable` is a distributed storage system to scale to petabytes of data across many nodes and is used in many google services @bigtable [p. 1]. However, it does not support a full relational model and instead provides a simple data model. Effectively, `Bigtable` uses a LSM-Tree architectute with memtables and SStables described like in @lsm_fig. 

`LevelDB` is an LSM-based key-value store developed by Google and released as open source in 2011 @lsm_storage_survey[p. 413]. It established the basic design of multiple levels (proposed by O'Neil et al. as $C_n$ components) and compaction strategies that are now standard in LSM-Tree implementations. @lsm_storage_survey. `RocksDB` is a fork of `LevelDB` developed by Facebook in 2012, with additional features and optimizations @lsm_storage_survey[p. 413]. 

`Apache Cassandra` is a highly scalable, distributed NoSQL database designed to handle large amounts of data across many node without a single point of failure @lsm_storage_survey[p. 415]. Every single partition in `Cassandra` is stored as a separate LSM-Tree @lsm_storage_survey[p. 415], allowing for high write throughput and fault tolerance at the same time. As we will see in TODO, LSM-Trees are slower for lookups in comparison to B-Trees, which is why `Cassandra` also supports secondary indexes, where the secondaty indexes are updated, if a record is found in the Memtable @lsm_storage_survey[p. 415]. 

Besides more traditional applications, LSM-Trees are also widely used in time-series databases. Those systems capture and store events in time-ordered data, ranging from IoT sensor data to financial transactions @indluxdata[p. 7-10]. `InfluxDB` is the most popular time-series database @dbengines_ranking and uses a storage engine similar to a LSM-Tree @infuxtsm. It also uses a write-ahead log and so called `TSM files` as on-disk components, which are similar to SSTables @infuxtsm. Similar to a traditional `Memtable`, `InfluxDB` uses a in-memory `Cache` which stores all recent writes to the write-ahead log and queries then merge from the `Cache` and the `TSM files` @infuxtsm.

