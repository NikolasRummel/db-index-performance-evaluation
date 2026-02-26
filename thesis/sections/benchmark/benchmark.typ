#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"

TODO: Section about Use cases OLTP, OLAP, WRITE HEAVY, READ HEAVY!

= Design and Implementation <design>

The goal of this chapter is to describe the design of the benchmark and the implementation of the storage manager, index structures and some highlights on the benchmark itself. The benchmark will be designed to evaluate the performance of different index structures under various workloads, and the implementation will be done in Go programming language.

== Requirements 
In order to design the benchmark, we can use typical software engineering practices and first define the requirements for the benchmark. These requirements will guide the design and implementation of the benchmark and ensure that it meets the goals of the thesis. The requirements can be categorized into functional requirements, which describe what the benchmark should do, and non-functional requirements, which describe how the benchmark should perform.

=== Functional Requirements
For the benchmark, the following index structures will be implemented and compared:
- *B-Tree:* A normal B-Tree where data is stored in both internal and leaf nodes should be implemented like described in @btree
- *B+-Tree:* A B+-Tree where data is only stored in linked leaf nodes based in order to compare the range query performance. Moreove because B+-Trees are the most common index structure used in DBMS.
- *LSM-Tree:* Since there is not enough time to implement a full LSM-Tree from scratch, an existing implementation will be used. For this, some evaluation will be done on existing Go libraries and then the best fitting one will be choosen to be used. 
- *No index:* As a baseline, a simple scan of the data without using any index will be implemented to compare the performance of the index structures against a full scan.

In order to compare the performance of these index structures and answering the research questions, the benchmark will consist of multiple tests (T1-T5) that will evaluate different aspects of the index structures under different workloads. The following will be designed to measure the following performance metrics:

+ *Point query lookup (T1):* This test measures the latency of retrieving a single value associated with a specific, randomly selected key. The benchmark executes a high volume of unique lookups against a pre-populated index to calculate average, median and the 95th percentile latency. This simulates typical OLTP workloads where fast access to individual records is critical. The output should be a box plot showing the distribution of latencies for each index structure.

+ *Range query lookup (T2):* Here, the benchmark evaluates the performance of range queries by measuring the time taken to retrieve all key-value pairs within a specified key range $[R_s;R_e]$. The test captures the latency of executing range queries of varying sizes (e.g., $10^6$, $10^7$, $10^8$ keys) to analyze how fast the index structures handle larger result sets, which is important for OLAP workloads. The output should be a line graph showing the latency of range queries as the size of the result set increases for each index structure.

+ *Write throughput over time (T3):* This test measures the write throughput of each index structure by continuously inserting new key-value pairs over a fixed duration while monitoring the number of insertions per second. This simulates write-heavy workloads especially as the dataset grows. The output should be a time series graph showing the write throughput over time for each index structure.

+ *Read heavy workload (T4):*  Here, the test simulates "realistic" workloads by executing a mix of read and write operations with a high read-to-write ratio (e.g., 95/5). Applications to this workload could be some E-commerce webshops or some social media like linkedin. It measures the latency of read operations over time to evaluate how well the index structures maintain read performance under a mostly read workload. The output should be a line graph showing the read latency over time for each index structure under the mixed workload. 

+ *Write heavy workload (T5):* Like in T4 and similar to T4, this test simulates a mixed workload but now with a high write-to-read ratio (e.g., 5/95). This could be realistic in some kind of database workload like for industial machines which stores sensor data or metrics in the database. It measures the latency of write operations over time to evaluate how well the index structures maintain write performance under a mostly write workload. The output should be a line graph showing the write latency over time for each index structure under the mixed workload.

+ *Memory usage (T5):* For all tests above, the memory usage of each index structure will be monitored and recorded to evaluate the memory efficiency of each index structure under the different workloads. The output should be bar charts showing the usage of each index structure for each test.

=== Non-Functional Requirements
In addition to the functional requirements, the benchmark should also meet the following non-functional requirements:

- *Reproducibility:* The benchmark should store its generated values and  reports in a way that allows for reproducibility of the results. This means that after re-running the benchmark with the same parameters, the same results should be obtained. If generated data is not deleted, this will be used in order to ensure reproducibility.

- *Fairness:* The benchmark should ensure that all index structures are tested under the same conditions and workloads to ensure a fair comparison. This includes using the same dataset, the same hardware, and the same configuration for each index structure. Since the LSM-Tree implementation is an external library, in the analysis of the results, the differences in implementation and optimizations will be taken into account to ensure a fair comparison.

- *Code Quality:* The implementation of the benchmark and the index structures should follow good software engineering practices, including modular design, clear documentation, and maintainable code. This will ensure that the code is easy to understand and modify if needed.


== Coding language and libraries
To implement the benchmark and the index structures, a programming language needs to be selected. The selection of the programming language will be based on criteria such as performance or how complex the language is to work with, since the time is limited for this project.

=== Programming Language
To select a programming language for the implementation of the benchmark the following methodology will be used:
+ *Selection Criteria:* A set of criteria will be defined to evaluate the suitability of different programming languages for the implementation of the benchmark. 
+ *Gathering of Candidate Languages:* A list of candidate programming languages will be gathered.
+ *Matrix for Decision:* A decision matrix will be created to evaluate the candidate languages based on the defined criteria.

==== Selection Criteria
The selection of a programming language for a this project requires a balance between low-level hardware access and development efficiency, since only one person is working on the project and the implementation of index structures and a benchmark can be complex. 
Therefore, the following criteria with weighting were considered for the selection of the programming language:
- *System Performance:* The language should provide low-level access to memory and hardware to allow for efficient implementation of index structures and the benchmark. 
- *Language Complexity:* The effort required to implement a prototype of index structures and benchmark should be manageable, so that the implementation can be completed within the time frame of the thesis.

- *Community and libraries:* The language should have a strong community and a good ecosystem of libraries. Also some #gls("DBMS") or similar projects implemented in the language should existst to ensure that it is suitable for database development.

- *Personal Experience :* Existing knowledge will be taken into account to ensure project completion within the time frame.

- *Learning Objectives:* Since this is a project for the university, the opportunity to learn a new programming language and gain experience with it will also be considered as a criterion for selection.


==== Gathering of Candidate Languages
To provide a justification for the selection, two methods were used to select languages which then were evaluated based on the criteria above:
+ *GitHub Repository Analysis* A search for "database" projects on GitHub revealed that the most "starred" and influential open-source storage engines are predominantly built using C, C++, Go, and Rust. 
+ *DB-Engines Ranking Evaluation:* The DB-Engines Ranking @dbengines_ranking, which measures the popularity and market adoption of almost 400 #gls("DBMS"), was consulted to identify the implementation languages of the world's most successful databases. Here, MySQL is written in C and C++, PostgreSQL in C, MongoDB in C++. TODO Cite https://www.tencentcloud.com/techpedia/134379 or search better sources.


==== Matrix for Decision
The candidate languages were scored from 1 (lowest) to 5 (highest) based on the criteria above:


#table(
  columns: (auto, 1fr, 1fr, 1fr, 1fr, 1fr, 1fr),
  inset: 6pt,
  align: horizon,
  [*Criterion*], [*Weight*], [*C*], [*C++*], [*Rust*], [*Go*], [*Java*],
  [System Performance], [0.25], [5], [5], [5], [4], [2],
  [Language Complexity], [0.20], [3], [2], [1], [5], [4],
  [Community & Ecosystem], [0.15], [5], [5], [4], [5], [5],
  [Personal Experience], [0.20], [2], [2], [1], [4], [5],
  [Learning Objectives], [0.20], [3], [3], [5], [5], [1],
  [*Total Score*], [1.00], [3.65], [3.45], [3.25], [*4.55*], [3.20],
)

TODO: Change values


==== Result
For this project, the Go programming language was choosen for the implementation of the index structures and the benchmark. 
Inspired by the C programming language, Go is a statically typed, compiled language that however also provides high-level features like garbage collection and built-in support for concurrency @golang[preface p. xii] @godocs. Go was created by Google since they were dealing more and more with complex software systems @golang[preface p. xiiii] and now is widely used in the industry #footnote[https://survey.stackoverflow.co/2025/technology#most-popular-technologies-language]. 
With Go being a modern language, it provides a good balance between performance and ease of development, which makes it a good choice for implementing the index structures and the benchmark. Languares like C++ and Rust may be more performant but are more complex to work with, which is why Go was choosen. Additionally, Go has a huge standard library and a large ecosystem of third-party libraries that can be used to facilitate the implementation @godocs. There are also some @DBMS like CockroachDB that are implemented in Go, which shows that it is a suitable language for database development @cockroachdb.  

=== Libraries
==== LSM-Tree Implementation
Pepple 
==== Plotting and Visualization
Gonum 




== Architectural Overview
#figure(caption: "Component Diagramm of the Benchmark", image(width: 4cm, "../../assets/comp.jpeg"))


#figure(
  caption: [ Architectural Overview of the Benchmark System.],
  cetz.canvas(length: 1cm, {
    import cetz.draw: *
    
    let box-style = (stroke: 0.8pt, radius: 2pt)
    let fill-bench = blue.lighten(95%)
    let fill-index = orange.lighten(95%)
    let fill-storage = green.lighten(95%)
    let fill-io = gray.lighten(95%)
    let fill-pebble = purple.lighten(95%)

    // --- BENCHMARK LAYER ---
    rect((0, 8.5), (12, 10), fill: fill-bench, ..box-style, name: "bench_layer")
    content("bench_layer", [*Benchmark* (Orchestrates all 3 Indexes)], size: 9pt)

    // --- INDEX INTERFACE ---
    rect((0, 6.8), (12, 7.8), fill: fill-index, ..box-style, name: "index_iface")
    content("index_iface", [*Index Interface* (Common API: Get/Insert/Range)], size: 9pt)

    // --- CUSTOM IMPLEMENTATIONS (Layered) ---
    rect((0, 4.8), (3.5, 6.0), fill: white, ..box-style, name: "bt_comp")
    content("bt_comp", [B-Tree \ (btree.go)], size: 8pt)
    
    rect((4.25, 4.8), (7.75, 6.0), fill: white, ..box-style, name: "bpt_comp")
    content("bpt_comp", [B+ Tree \ (bptree.go)], size: 8pt)

    // --- EXTERNAL WRAPPER ---
    rect((8.5, 4.8), (12, 6.0), fill: fill-pebble, ..box-style, name: "lsm_comp")
    content("lsm_comp", [LSM-Tree \ (Pebble Wrapper)], size: 8pt)

    // --- SHARED ENGINE (Custom Only) ---
    rect((0, 2.8), (7.75, 4.0), fill: fill-storage, ..box-style, name: "shared_engine")
    content("shared_engine", [*Shared Tree Engine* \ (Generic Logic)], size: 8pt)

    // --- STORAGE LAYERS ---
    rect((0, 0.8), (7.75, 2.0), fill: fill-io, ..box-style, name: "pager_layer")
    content("pager_layer", [*Custom Pager* \ (LRU Cache + Disk I/O)], size: 8pt)
    
    rect((8.5, 0.8), (12, 4.0), fill: fill-pebble, ..box-style, name: "pebble_internal")
    content("pebble_internal", [*Pebble Internal* \ (Memtable, SSTables, \ Internal Cache)], size: 8pt)

    // --- PHYSICAL DISK ---
    rect((0, -1), (12, 0), fill: white, stroke: (dash: "dashed"), name: "disk")
    content("disk", [OS File System (.bt, .bpt, .lsm files)])

    // --- CONNECTORS ---
    line("bench_layer.south", "index_iface.north", mark: (end: "stealth"))
    
    // Custom Tree path
    line((1.75, 6.8), (1.75, 6.0), mark: (end: "stealth"))
    line((6.0, 6.8), (6.0, 6.0), mark: (end: "stealth"))
    line((1.75, 4.8), (1.75, 4.0), mark: (end: "stealth"))
    line((6.0, 4.8), (6.0, 4.0), mark: (end: "stealth"))
    line((3.87, 2.8), (3.87, 2.0), mark: (end: "stealth"))
    line((3.87, 0.8), (3.87, 0), mark: (end: "stealth"))

    // Pebble path
    line((10.25, 6.8), (10.25, 6.0), mark: (end: "stealth"))
    line((10.25, 4.8), (10.25, 4.0), mark: (end: "stealth"))
    line((10.25, 0.8), (10.25, 0), mark: (end: "stealth"))
  })
) <fig-component-arch-fixed>
== Index implementations

#figure(
  caption: "Ein Stück Quellcode",
  sourcecode[```go
    type Index interface {
        Insert(key int64, value []byte) error
        Get(key int64) ([]byte, error)
        Delete(key int64) error
        Range(start, end int64) (Iterator, error)
        Close() error
    }

    type Iterator interface {
        Next() bool
        Key() int64
        Value() []byte
        Close() error
    }
```],
)

== Storage Manager Implementation
=== Pager 
=== LRU Cache 

=== Generic Tree Implementation
==== Insertion algorithm
==== Point query algorithm
==== Range query algorithm

=== B-Tree Implementation
=== B+-Tree Implementation
=== LSM-Tree Implementation

== Benchmark Design
=== Deterministic data generation
=== Workload design
==== T1: Point query lookup
==== T2: Range query lookup
==== T3: Write throughput over time
==== T4: Mixed workload
==== T5: Memory usage

== Generating reports and visualizations
