#import "@preview/clean-dhbw:0.5.0": gls

= Conclusion <conclusion>
This thesis provided a comparison of B-Trees, B+-Trees, and LSM-Trees in form of empirical benchmarks. After looking at some theoretical properties of all three index structures, a concept on how to implement and benchmark them was developed. 
The implementation of the B-Tree and B+-Tree was done from scratch, while for the LSM-Tree, the Pebble library was used. The benchmarks were designed to evaluate the performance of each index structure under different workloads. As theoretically expected, the B+-Tree outperformed the standard B-Tree in range query scenarios, while the LSM-Tree performed best in write-heavy workloads. The results clearly demonstrate the architectural differences between the structures and therefore the importance on choosing the right index structure or in general the right #gls("DBMS") for the right use case.

== Answering the Research Questions
Based on the analysis in @evaluation, the research questions formulated in @research_questions can be answered as follows:

*RQ1: How do B-Trees, B+-Trees, and LSM-Trees compare in terms of query speed?*
B+-Trees offer the highest random lookup speed due to their optimized tree height. Standard B-Trees are slower because internal nodes store values, leading to a deeper structure. LSM-Trees are the slowest for point queries, as a lookup may require searching across multiple levels and components. For range queries, B+-Trees significantly outperform B-Trees due to their linked leaf nodes, which allow for efficient linear traversal. On the other side, LSM-Trees perform very well for write-intensive workloads, resulting in much higher (22-55x more) write throughput than B-Trees, but reads are much slower as the tradeoff.

*RQ2: How significant is the performance gap between B-Trees and B+-Trees during range queries?*
In the benchmarks, the B+-Tree, which can be further optimized, was approximately twice as fast as the B-Tree for range scans as seen in T2. This is attributed to the B+-Tree's linked leaf list, which allows for linear traversal, whereas the B-Tree must perform a more complex in-order traversal using a stack. The greater the dataset range and the more leaf nodes involved, the bigger the performance gap becomes. This is why B+-Trees are the dominant index structure in relational databases, as they provide superior performance for both point and range queries.

*RQ3: Which index structure should you choose for a write- or read-heavy workload?*
For read-heavy or balanced workloads, the B+-Tree is the superior choice due to its consistent and low latency. For write-heavy workloads where ingestion throughput is the primary concern, the LSM-Tree is clearly preferable, provided that the application can tolerate higher read times if sometimes reads are necessary. 

== Critical Reflection 
While the results are conclusive, some limitations must be considered. The comparison between a mature, optimized LSM implementation  with a lot of features like `Pebble` and a relatively simple, custom-built B-Tree implementation may not be entirely fair. A real B+-Tree implementation (e.g., from InnoDB) would likely show more realistic performance differences. 

In addition, the B-Trees are not fully implemented. A `delete()` method is missing, and a lot is not optimized or left out for simplicity and time constraints. For example, in the generic tree implementation the `nextLeaf` pointer field is reserved on every page but unused for B-Trees, leading to an overhead of 4 bytes. In addition, in both B-Tree implementations, the nodes can have internal fragmentation. Updates with larger values do not reclaim the space of the previous version, leading to fragmentation, therefore wasted space and in the end to worse performance. 

Another point which should be mentioned is that the choice of the `SyncInterval` being `500` in the buffer manager (`Pager` component) from @pager was made without an empirical justification. A more thorough analysis of the impact of different synchronisation intervals on write throughput would have been beneficial to better compare the real performance differences between B-Trees and LSM-Trees. However the main differences between the two structures are so significant that this would not change the overall conclusion too much.

Besides all these limitations, for the purpose of this thesis, which is to compare the three index structures, the implementation is sufficient to show the expected performance differences and to answer the research questions. 

== Outlook <outlook>
To enhance the findings of this thesis, several potential avenues for future research and improvement can be explored beyond fixing the limitations mentioned in the critical reflection:

1. *Concurrency Control:* The current implementation focuses on single-threaded performance. Future work could integrate multi-threading to evaluate how well each index structure handles concurrent access patterns, which is critical for real-world database systems.
2. *Custom LSM-Tree implementation:* Implementing a custom LSM-Tree from scratch, similar to the B-Tree and B+-Tree, would allow for a fair comparison and a deeper understanding of the implementation of LSM-Trees. 
3. *Explore more modern index structures:* Investigating newer index structures that have emerged in recent years could provide insights into how the field is evolving and whether there are alternatives that offer better performance for specific workloads. 
4. *Real-World Database Comparison:* Instead (or in addition to the custom implementations) of the B-Tree and B+-Tree, a comparison with real-world database systems that use these structures could provide more practical insights into their performance under realistic workloads.
5. *Creating a complete #gls("DBMS"):* Moreover, the current code could be used to build a complete #gls("DBMS"), including a query processor, transaction management, and other features. This would be a very interesting project to further explore all topics regarding databases.


In summary, there are many promising directions for future research that could build upon this work, to get a deeper understanding of index structures or to explore other related topics in the field of database systems.
