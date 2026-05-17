#let glossary-entries = (
  (
    key: "Bloom",
    short: "Bloom filter",
    description: "Bloom filters are used to quickly check in $O(1)$ if an element is not present in a set. They are a space-efficient probabilistic data structure to reduce the number of disk accesses needed to find a key in a memtable or SSTable of a LSM-Tree.",
  ),
  (
    key: "writeamplification",
    short: "write amplification",
    description: "Write amplification is a phenomenon where the amount of data written to the storage media is greater than the amount of data intended to be written. The goal is to minimize write amplification, as it can lead to increased wear on storage devices and unnecessary writes.",
  ),
  (
    key: "B-Tree",
    short: "B-Tree",
    description: "A self-balancing tree data structure that maintains sorted data and allows searches, sequential access, insertions, and deletions in logarithmic time.",
  ),
  (
    key: "B+-Tree",
    short: "B+-Tree",
    description: "A variation of the B-Tree where all data is stored in the leaf nodes, and internal nodes only store keys to guide the search. The leaf nodes are also linked to allow efficient range queries.",
  ),
  (
    key: "LSM-Tree",
    short: "LSM-Tree",
    description: "An index structure designed for high-performance writes by buffering updates in memory and periodically merging them into sorted files on disk.",
  ),
  (
    key: "Memtable",
    short: "Memtable",
    description: "An in-memory data structure (often a SkipList or B-Tree) used in LSM-Trees to buffer incoming writes before they are flushed to disk as SSTables.",
  ),
  (
    key: "SSTable",
    short: "SSTable",
    description: "Sorted String Table. A file format used in LSM-Trees to store sorted key-value pairs on disk. They are immutable once written.",
  ),
  (
    key: "Compaction",
    short: "Compaction",
    description: "The process in LSM-Trees of merging multiple SSTables into a single, larger SSTable, removing deleted or overwritten keys to reclaim space and maintain read performance.",
  ),
  (
    key: "WAL",
    short: "WAL",
    long: "Write-Ahead Log",
    description: "A log file used to provide durability by recording all changes before they are applied to the main data structures. In case of a crash, the log can be replayed to recover lost data.",
  ),
  (
    key: "Page",
    short: "page",
    description: "The fixed-size unit of data transfer between the DBMS and storage, typically 4 KB or 8 KB. It is the atomic unit for buffer management.",
  ),
  (
    key: "Block",
    short: "block",
    description: "A contiguous sequence of sectors on a storage device, often used as the smallest unit of I/O by the operating system.",
  ),
  (
    key: "Sector",
    short: "sector",
    description: "The smallest addressable unit on a physical disk drive, traditionally 512 bytes or 4 KB.",
  ),
  (
    key: "Node",
    short: "node",
    description: "A fundamental unit of a tree data structure, containing keys and pointers to other nodes.",
  ),
  (
    key: "Leaf Node",
    short: "leaf node",
    description: "A node in a tree structure that has no children. In B+-Trees, all data pointers are stored in the leaf nodes.",
  ),
  (
    key: "Internal Node",
    short: "internal node",
    description: "A node in a tree structure that has child nodes. It is used to guide the search to the appropriate leaf node.",
  ),
  (
    key: "Wearleveling",
    short: "wear leveling",
    description: "A technique used in SSDs to distribute write and erase cycles evenly across the memory cells to prolong the lifespan of the device.",
  ),
  (
    key: "gc",
    short: "garbage collection",
    description: "In the context of SSDs, garbage collection is the process of reclaiming space by erasing blocks that contain invalid or outdated data. In general, it is a form of automatic memory management that frees up memory that is no longer in use.",
  )
)

#let acrolist-entries = (
  (
    key: "API",
    short: "API",
    long: "Application Programming Interface",
  ),
  (
    key: "HTTP",
    short: "HTTP",
    long: "Hypertext Transfer Protocol",
  ),
  (
    key: "DBMS",
    short: "DBMS",
    long: "Database Management System",
  ),
  (
    key: "SQL",
    short: "SQL",
    long: "Structured Query Language",
  ),
  (
    key: "HDD",
    short: "HDD",
    long: "Hard Disk Drive",
  ),
  (
    key: "SSD",
    short: "SSD",
    long: "Solid State Drive",
  ),
  (
    key: "LSM",
    short: "LSM",
    long: "Log-Structured Merge-tree",
  ),
  (
    key: "RAM",
    short: "RAM",
    long: "Random Access Memory",
  ),
  (
    key: "OLTP",
    short: "OLTP",
    long: "Online Transactional Processing",
  ),
  (
    key: "OLAP",
    short: "OLAP",
    long: "Online Analytical Processing",
  ),
  (
    key: "AI",
    short: "AI",
    long: "Artificial Intelligence",
  ),
  (
    key: "SLC",
    short: "SLC",
    long: "Single-Level Cell",
  ),
  (
    key: "MLC",
    short: "MLC",
    long: "Multi-Level Cell",
  ),
  (
    key: "TLC",
    short: "TLC",
    long: "Triple-Level Cell",
  ),
  (
    key: "QLC",
    short: "QLC",
    long: "Quad-Level Cell",
  ),
  (
    key: "NAND",
    short: "NAND",
    long: "Not-AND",
  ),
  (
    key: "LRU",
    short: "LRU",
    long: "Least Recently Used",
  ),
  (
    key: "MRU",
    short: "MRU",
    long: "Most Recently Used",
  ),
  (
    key: "FIFO",
    short: "FIFO",
    long: "First-In, First-Out",
  ),
  (
    key: "OS",
    short: "OS",
    long: "Operating System",
  ),
  (
    key: "GB",
    short: "GB",
    long: "Gigabyte",
  ),
  (
    key: "CSV",
    short: "CSV",
    long: "Comma-Separated Values",
  ),
  (
    key: "ACID",
    short: "ACID",
    long: "Atomicity, Consistency, Isolation, Durability",
    description: "A set of properties that guarantee reliable processing of database transactions.",
  ),
  (
    key: "IPO",
    short: "IPO",
    long: "Input-Process-Output",
    description: "A model that describes the typical structure of a program or system which follows this 3-step process.",
  ),
  (
    key: "CRUD",
    short: "CRUD",
    long: "Create, Read, Update, Delete",
    description: "The four basic operations of persistent storage.",
  ),
  (
    key: "DDL",
    short: "DDL",
    long: "Data Definition Language",
    description: "A subset of SQL used to define and modify database structures such as tables, indexes, and schemas.",
  ),
  (
    key: "DML",
    short: "DML",
    long: "Data Manipulation Language",
    description: "A subset of SQL used to insert, update, delete, and retrieve data from the database.",
  ),
  (
    key: "I/O",
    short: "I/O",
    long: "Input/Output",
    description: "Operations that transfer data between a computer and external storage devices or peripherals.",
  ),
  (
    key: "NoSQL",
    short: "NoSQL",
    long: "Not Only SQL",
    description: "A class of database systems that do not use the traditional relational model, often used for handling large volumes of unstructured or semi-structured data.",
  ),
)
