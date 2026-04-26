#import "@preview/clean-dhbw:0.4.0": gls

= Results and Analysis <evaluation>

In this chapter, the results of the benchmarks will be presented and analyzed. 


== Benchmark Setup 
The benchmarks were conducted on a 2021 MacBook Pro machine with the specification @specs.


#figure(
  caption: [Benchmark Machine Specifications],
  text(size: 0.9em, 
    table(
      columns: (auto, auto),
      inset: 5pt,
      align: (right, left), 
      table.header([*Component*], [*Specification*]),
      [Processor], [Apple M1 Pro (10-Core)],
      [Memory], [32 GB Unified Memory],
      [Storage], [Internal NVMe SSD],
      [OS], [macOS 16.3],
      [Size], [16-inch],
      [File System], [APFS],
      [Go Runtime], [go1.22.x (arm64)],
    )
  )
)<specs>

As configuration of the benchmark, the following index parameters were used:

+ B-Tree with node/page size of  
 - 4 kb
 - 8 kb
 - 16 kb

+ B+-Tree with node/page size of  
 - 4 kb
 - 8 kb
 - 16 kb

+ LSM-Tree with a memtable size of 
 - 16 mb
 - 32 mb
 - 64 mb 

For B-Trees, the cache is configured to hold 4096 pages, so 4kb pages result in a cache size of 16 mb, 8 kb pages result in a cache size of 32 mb, and 16 kb pages result in a cache size of 64 mb. This allows for a fair comparison between the B-Tree and LSM-Tree configurations, as they have the same size of in-memory data. 

== Results and Analysis of Test 1

For T1 5 million records were first inserted sequentially to populate each index. Subsequently, a workload of 2 million random point queries was executed. This is enough to fill the cache and memtable of each index, mimicking a realistic workload. The results of this test are shown in the following charts, where the first chart shows a boxplot of response times and the second chart shows the throughput in queries per second.

#figure(
  caption: [T1: Point Query P95 Response Time],
  image(width: 100%, "../../assets/results/t1boxplot.png")
)<t1boxplot>

=== B-Tree vs. B+-Tree
The data in @t1boxplot reveals a clear performance gap between the standard B-Tree and the B+-Tree. The B+-Tree (4k) achieves significantly lower latency because it stores all data records in the leaf nodes, while internal nodes contain only keys and pointers. As established in Section @b-plus-disk-mapping, this design increases the fan-out—the number of children per node—thereby reducing the total tree height $h$. The B-Tree 4k has a height of 6 while the B+-Tree (4k) has a height of 4. In contrast, the standard B-Tree (Section @btree) stores values at every level, which consumes space in internal nodes and forces a deeper tree structure, requiring more page fetches for each random lookup.

The data in @t1boxplot shows that B+-Trees are faster then normal B-Trees. Comparing the smallest configurations of both index types, the B+-Tree (4k) with a median time of 2.5 $mu s$ its about two times faster than the B-Tree (4k) with a median time of 4.75 $mu s$. As explained in @b-plus, B+-Trees store more keys per node, which lead to a smaller tree. During this test, the B-Tree 4k has a height of 6 while the B+-Tree (4k) has a height of 4. This means for a point query, the B-Tree (4k) needs to fetch up to 6 pages from disk, while the B+-Tree (4k) only needs to fetch 4 pages, which results in a significant performance improvement. This also applied for larger page sizes, where the B+-Trees are consistently faster than the B-Trees.


 
#figure(
  caption: [T1: Point Query Throughput],
  image(width: 100%, "../../assets/results/t1throughput.png")
)<t1throughput>

=== LSM-Tree performance
As seen both in @t1boxplot and @t1throughput, the LSM-Tree are slower compared to the B-Trees. Also the LSM-Tree is optimizised since with it being Pebble, it is a mature implementation with many optimizations, whereas the B-Trees are simple implementations with potential for further optimizations. The main issue with LSM-Trees is that as shown in @lsm_oneil, a lookup must search within multiple components, which is not fast compared to the B-Trees.

== Results and analysis of T2
Here, each index is inserted with 5 million records. Then, range queries of different sizes are executed, starting with a range size of 4k and doubling the range size until reaching 5 million. In the following, the results will be discussed for the different index types and afterwards, the impact of different page sizes for B-Trees will be analyzed.

#figure(
  caption: [T2: Range Query Performance of all index types],
  image(width: 100%, "../../assets/results/t2rangeall.png")
)<t2rangeall>

In @t2rangeall, we clearly see the performance gap between B-Trees and B+-Trees for range queries. The B+-Trees are significantly faster than the B-Trees, especially if the range sizes becoming larger. The reason here is the linked list of leaf nodes in B+-Trees, which allows to reach the next node faster, in comparison to B-Trees. In @rq we saw that the normal B-Tree must perform a in-order traversal, which results in this performance difference. 

To answer the second research question (RQ2), we see that with the current implementation, which probably can be further optimized, the B+-Tree is about 2 times faster than the B-Tree for range queries. The bigger the range scan is, the bigger the performance gap becomes. This is why for a real #gls("DBMS") B+-Trees are the most commonly used index structure in database systems.

Another interesting point is to look at different configuration of node/page sizes for B-Trees. Here we look at the performance of B-Trees with 4k, 8k and 16k page sizes, the differences within each are the same for B+-Trees.

#figure(
  caption: [T2: Range Query Performance withtin different sizes of nodes/pages for B-Trees],
  image(width: 100%, "../../assets/results/t2rangeb.png")
)<t2rangeb>


Here we see that bigger page sizes result in better performance for range queries. This is because bigger page sizes allow to store more keys per node, which results in a smaller tree and thus less leaf page fetches for range queries. The following table shows the structural differences between the different configurations of B-Trees for 5 million records.
#figure(
  table(
    columns: (auto, auto, auto, auto),
    inset: 10pt,
    align: horizon,
    [*Page Size*], [*Tree Height*], [*Leaf Nodes*], [*Avg. Records/Leaf*],
    [4k], [6], [333333], [~15],
    [8k], [5], [172413], [~29],
    [16k], [4], [86206], [~58],
  ),
  caption: [Structural Comparison of B-Tree configurations for 5M records],
) <btree-stats>

Since now the configurations with bigger pages, during the range scan, the buffer manager needs to fetch less leaf pages, which results in better performance we see in @t2rangeb. This is one reason why for instance 'InnoDB' uses 16k pages by default, as we saw in @bplus_practice. 

== Results and analysis of T3
For T3, 5 million records where continuously inserted into each index, while the throughput was measured. The results are shown in the following chart. 
#figure(
  caption: [T3: Write throughput performance of all index types],
  image(width: 100%, "../../assets/results/t3.png")
)<t3>

As expected, the LSM-Tree outperforms the B-Trees by a factor of roughly 3. In the beginning, all LSM-Trees are even faster, since the memtable is not yet full and thus all writes are done in-memory, which is very fast. As the memtable fills up, the performance of the LSM-Tree decreases, but it still outperforms the B-Trees. The B-Trees have a much lower write throughput, since for each insert, the tree needs to be updated on disk, which requires multiple page fetches and writes for each insert. This is especially problematic for random inserts, which require more page fetches and writes due to the need to maintain the tree structure.

In comparison to B-Trees, the throughput of the LSM-Tree is also more unstable, since we write on the memtable until it is full, which results in a sudden drop in performance. After the memtable is full, the LSM-Tree needs to flush the memtable to disk and merge it with the on-disk components, which also results in a drop in performance. This pattern is repeated throughout the test, which results in the unstable performance of the LSM-Tree we see in @t3lsm.

#figure(
  caption: [T3: Performance of LSM-Tree during the test],
  image(width: 100%, "../../assets/results/t3lsm.png")
)<t3lsm>


Furthermore, the frequency of these performance drops is inversely proportional to the MemTable size. As seen in @t3lsm, the 16MB LSM-Tree produces a much higher frequency of flushes compared to the 32MB and 64MB versions. By increasing the MemTable capacity, the tree can buffer more incoming writes before reaching the capacity threshold that triggers a flush to disk. This realationship could be expressed as $f_"flush" approx frac(v_"write", S_"mem")$ where $f_"flush"$ represents the frequency of the performance drops, $v_"write"$ is the ingestion rate, and $S_"mem"$ is the MemTable size.

Looking at the 16MB and 32MB LSM-Tree, we can see that at some point there are drops in performance dropping down to 100.000-200.000 ops/sec. The reason here are so called "write stalls", which occur when the memtable is full and the LSM-Tree needs to flush the memtable to disk, but the flush process is not yet finished and thus the LSM-Tree cannot accept new writes until the flush is finished @pebble_readme. To mitigate this problem, the throughput is being reduced, which is why we see in @t3lsm the drops in performance. The 64MB LSM-Tree does not have these drops in performance, since it has a larger memtable size and thus can buffer more writes before reaching the capacity threshold that triggers a flush to disk.


== Results and analysis of T5 

