
#import "@preview/clean-dhbw:0.4.0": *

== Practical Relevance and Industrial Application <practice>

While choosing an index structure to design a #gls("DBMS"), the choice between B+ Trees and LSM-Trees is primarily dictated by the expected ratio of read-to-write operations and the target storage medium. While B+ Trees remain the foundation for traditional transactional systems, LSM-Trees have emerged as the dominant architecture for distributed, write-heavy, and cloud-native workloads. (S3)

=== B+ Tree-Based Systems 
Traditional relational #gls("DBMS") like PostgreSQL and MySQL primarily utilize B+ Tree index structures to maintain ACID compliance and optimize for OLTP workloads @mysql_innodb. 

`MySQL` was first released in 1995 and has become a widely used #gls("DBMS") with over 1 billion installations @mysql_release. The default storage engine for MySQL, `InnoDB`, data is organized into B-trees @mysql_physical_structure. Since the data is stored in the leaf nodes, `InnoDB`'s B-tree is technically a B+-Tree. To optimize performance, `InnoDB` uses 16KB pages by default. For sequential inserts, it keeps pages 15/16 full to allow for future growth. For random inserts, the fill factor varies between 50% and 93.75%. If a page's data drops below the 50%  due to deletions, `InnoDB` automatically merges neighboring pages to prevent fragmentation and optimize the on-disk footprint.@mysql_physical_structure

`PostgreSQL` is a open soruce relational #gls("DBMS") that uses a special B-Tree proposed by Lehman, Philip L. and Yao, s. Bing @postgres_btree_paper, which optimizes the performance of concurrent transactions. Like in Mysql, the B-tree is organized as a B+-tree, where the leaf nodes contain the actual data records, and the internal nodes contain pointers to the leaf nodes. 

`SQLite` was initially released in 2005 and has since become the most widely deployed database engine in the world @sqlite_most_deployed. It is written in `C` and was designed to be integrated in embedded systems like mobile applications, therefore being very lightweight @sqlite_general and easy to use. 
Based on SQLite's documentation @sqlite_fileformat, it stores data in a single file and uses two types of B-trees: one for tables and another for indexes. The B-tree for tables is used to store the actual data, while the B-tree for indexes is used to store the index entries.
The B-tree for tables is organized as a B+-tree, where again the leaf nodes contain the actual data records, while the internal nodes only contain pointers to the leaf nodes. 

