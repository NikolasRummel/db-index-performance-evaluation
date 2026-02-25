#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"


= Index Structures in DBMS <index>

== Motivation for Database Index Structures
When the stored data in a #gls("DBMS") grows, it gets more and more important to efficiently queiry the data. 

Imagine the szenario of a user searching for movies and series of a specific actor at Netflix. With approx 8000 titles available @statista_netflix, a scan though all titles to find the one maching the search query would take a long time. 

```sql
SELECT * FROM titles WHERE actor = 'Tom Hanks';
```

With a simple scan of the data, this query would have a performance of O(n), because in the worst case, all titles have to be checked.
To optimize this search prozess, it would make sense to just look at titles where the actor is 'Tom Hanks'. This can be archieved by using an index structure, which allows to find the relevant data much faster.

An index is a data stucture allowing to quickly locate the data we are looking for, without having to scan all the data. We can define the index on a specific attribute A, for instance on our example on the actor attribute @dbsystems_complete[P.~350]. This index will then speed up queries where A should match a specific value, like in our example above (A='Tom Hanks'). A very fast index would be a Hashmap, which could allow to find the title in O(1) time. However, which data structure is used for the index however depends on the use case and workload, which will be discussed in the following chapters.

== Types of Database Index Structures

As mentoned, there are different data structures which can be used as index structures in a database system. In the following, the most common index structures will be explained.


=== Search Tree-Based Index Structures <search-trees>
#let p = $p$
In order to understand the most common data structure for DBMS, the B+-Tree @elmasri2016 [p. 618], we will start with standard search trees. 


A search tree is a type of data structure used to organize and manage data in a way that allows for efficient searching. It is defined with an order #p so that each node contains at most $p - 1$ keys and $p$ pointers in the sequence $<P_1, K_1, P_2, K_2, ..., P_(q-1), K_(q-1), P_q>$, where $q <= p$. Each $K_i$ is a key which we are searching for in our tree data structure, whereas each $P_i$ represents the pointers to child nodes (including NULL pointers) @elmasri2016 [p. 618]. In other words, a node consists of values and pointers to other nodes. Each node have one more pointer than values since the pointers represent the subtrees between the values.

In order to be a search tree, this tree must satisfy the following constraints  @elmasri2016 [p. 618]:

1. In each node, the $K_i$ values are ordered such that $K_1 < K_2 < dots < K_(p-1)$ holds.
2. For all values $X$ in the subtree rooted at node $P_i$, the following conditions hold:
  - For $1 < i < q$: $K_(i-1) < X < K_i$
  - For $i = 1$: $X < K_1$
  - For $i = q$: $K_(q-1) < X$

In summary, this means that for each node in the tree, all keys in the left subtree are less than the key in the node, and all keys in the right subtree are greater. 


A search tree can then be visualized as follows:
#figure(
  caption: [B-Tree structure of order $p=3$ with extended right-heavy branch],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    
    rect((-0.7, 0.3), (0.7, -0.3), ..node-style, name: "n10")
    content("n10", [*10 | 20*])
    
    rect((-2.4, -1.2), (-1.0, -1.8), ..node-style, name: "n5")
    content("n5", [*5 | -- *])
    line((rel: (-0.3, 0), to: "n10.south"), "n5.north", ..edge-style)
    
    rect((-0.7, -1.2), (0.7, -1.8), ..node-style, name: "n15")
    content("n15", [*12 | 15*])
    line("n10.south", "n15.north", ..edge-style)
    
    rect((1.0, -1.2), (2.4, -1.8), ..node-style, name: "n25")
    content("n25", [*25 | 30*])
    line((rel: (0.3, 0), to: "n10.south"), "n25.north", ..edge-style)
    
    rect((2.0, -2.7), (3.4, -3.3), ..node-style, name: "n35")
    content("n35", [*35 | 40*])
    line("n25.south", "n35.north", ..edge-style)

    rect((3.0, -4.2), (4.4, -4.8), ..node-style, name: "n45")
    content("n45", [*45 | 50*])
    line("n35.south", "n45.north", ..edge-style)
  })
) <unbalanced-tree>

In this example, the search tree has a degree of $p=2$, meaning that each node can have at most one key and two child pointers. We see that the keys are organized in such a way that for each node, all keys in the left subtree are less than the key in the node, and all keys in the right subtree are greater.

==== Lookup <search-tree-lookup>
A lookup operation in a B-Tree is similar to a normal search tree described in @search-tree-lookup.
Now, to search for a specific key in the tree, for instance 35, we start at the root node (10) and follow the pointers if our search key is less than or greater than the key in the node. In this case, since 35 is greater than 10, we follow the right pointer to node 20. We repeat this process until we either find the key or reach a leaf node.

==== Performance Considerations

In our case from @unbalanced-tree, the search tree is unbalanced, meaning that not all paths from the root node to all leafes have the same length @dbsystems_complete[p. 634]. Therefore, we can see that we almost searched all nodes to find the key 35. In the worst case, the tree could basically just be a linked list, where a search would result in $O(n)$ time complexity. With this there would not be any advantage of using a search tree over kust scanning the data. In order to avoid this problem, it makes sence to use a balanced search tree.

=== B-Trees
To ensure that a search tree stays balanced, we can use a B-Tree. They where first described by Bayer and McCreight in 1972 @btree_original and are widely used in database systems both releational and non-relational @kleppmann[p. 80].
B-Trees are search trees with some additional contraints to ensure that the tree remains balenced @elmasri2016 [p. 619].
However, inserting and deletion of keys is more complex due to the need to maintain balance. In this section, we will also describe the lookup, insertion and deletion operations in a B-Tree in a low level of detail. A detailed implementation of these will be discussed in @design. 

TODO: ALSO: Complete index doesnt fit in memory!! 
-> original paper btree 

TODO2: Better Btree images, better examples with more data.


#figure(
  caption: [Balenced version of the previous tree structure in @unbalanced-tree],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    
    rect((-0.6, 0.3), (0.6, -0.3), ..node-style, name: "root")
    content("root", [*25 | -- *])
    
    rect((-2.2, -1.2), (-1.0, -1.8), ..node-style, name: "L1")
    content("L1", [*12 | -- *])
    line((rel: (-0.2, 0), to: "root.south"), "L1.north", ..edge-style)
    
    rect((1.0, -1.2), (2.2, -1.8), ..node-style, name: "R1")
    content("R1", [*35 | 45*])
    line((rel: (0.2, 0), to: "root.south"), "R1.north", ..edge-style)
    
    rect((-3.4, -2.7), (-2.4, -3.3), ..node-style, name: "leaf1")
    content("leaf1", [*5 | 10*])
    
    rect((-1.8, -2.7), (-0.8, -3.3), ..node-style, name: "leaf2")
    content("leaf2", [*15 | 20*])
    
    line((rel: (-0.3, 0), to: "L1.south"), "leaf1.north", ..edge-style)
    line((rel: (0.3, 0), to: "L1.south"), "leaf2.north", ..edge-style)

    rect((0.2, -2.7), (1.2, -3.3), ..node-style, name: "leaf3")
    content("leaf3", [*30 | -- *])
    
    rect((1.6, -2.7), (2.6, -3.3), ..node-style, name: "leaf4")
    content("leaf4", [*40 | -- *])

    rect((3.0, -2.7), (4.0, -3.3), ..node-style, name: "leaf5")
    content("leaf5", [*50 | -- *])
    
    line((rel: (-0.4, 0), to: "R1.south"), "leaf3.north", ..edge-style)
    line("R1.south", "leaf4.north", ..edge-style)
    line((rel: (0.4, 0), to: "R1.south"), "R1.south", "leaf5.north", ..edge-style)
  })
) <balanced-tree>

The constraints for a B-Tree of order $p$ are as follows @elmasri2016[p. 619] @dbsystems_complete[pp. 634-635]:
1. Like in normal search trees, we have a alternating sequence of keys and pointers in each node. However, now a B-Tree stores the values on the disc, so each entry need also a record pointer to the actual data record. This result in a Node structure of $<P_1, (K_1, "RP"_1), P_2, (K_2, "RP"_2), ..., P_(q-1), (K_(q-1), "RP"_(q-1)), P_q>$, where each $"RP"_i$ is the record pointer to the actual data record and $q <= p$.
2. Like in @search-trees, the keys in each node are ordered such that $K_1 < K_2 < dots < K_(q-1)$ holds.
3. All search key values $X$ within a subtree pointed by $P_i$ are bounded by the keys of the parent node. This ensures that subtrees contain only values from its parents key space. More formally, for all values $X$ in the subtree rooted at node $P_i$, the following conditions hold @elmasri2016 [p. 619]:
  - For $1 < i < q$: $K_(i-1) < X < K_i$
  - For $i = 1$: $X < K_i$
  - For $i = q$: $K_(i-1) < X$
4. To not end in a linked list in a node itself, each node has at most $p$ tree pointers.
5. All internal nodes (except the root and leaves) have at least $ceil(p/2)$ tree pointers to ensure some kind of density to avoid wasting space. However, the root node has at least two tree pointers if it is not a leaf node meaning its the only node in the tree.
6. All leaf nodes tree pointers $P_i$ are NULL and appear in the same level. This ensures that the tree is balanced and we get a guaranteed read performance of $O(h)$ with $h=log_p n$, where $n$ is the number of keys in the tree @intro_algorithms [p. 505]


==== Lookup 
A lookup operation in a B-Tree is similar to a normal search tree described in @search-tree-lookup @kleppmann[p. 80]. We start at the root node and compare the search key with the keys in the node. If we find a match, we return the corresponding record pointer to the disc and read from there the according value. If not, we follow the appropriate pointer to the child node based on the key comparison. We repeat this process until we either find the key or reach a NULL pointer in a leaf node, meaning the search key is not present in the tree.

==== Insertion 
Insertion might seem trivial at first, if there is some space in a node, insert the new key in the correct position to maintain the order. However as described Garcia-Molina et al. @dbsystems_complete [pp. 640-641], if the node is full, we need to split the node into two nodes and promote the middle key to the parent node to maintain the B-Tree properties. This process may propagate up to the root node, potentially increasing the height of the tree @btree_original[p. 178].

==== Deletion 
Again, deletion is most likely to be more complex than just removing the key from the node. First, it performes a lookup to find the key to be deleted @dbsystems_complete[p. 642] @btree_original[p. 190]. If the key is found in a leaf node, it can be removed directly. However, if the key is in an internal node, we again need to maintain the B-Tree properties described in @dbsystems_complete[p. 643] @btree_original[pp 180-182].

==== Range Query
Rangies Querys are a common operation in database systems, where we want to retrieve all keys within a specific range, for instance all titles between 'A' and 'D' @dbsystems_complete[p. 639]. For this example, we would do a lookup for the start key 'A' and then traverse the leaf nodes sequentially until we reach the end key 'D' with depth first in order walk. However, this will traverse all nodes in the range, which can be inefficient if the range is large. B+-Trees address this issue with leaf node chaining @, which will be explained in the next section.

=== B+-Trees
B+-Trees are a variant of B-Trees, fixing the problem of inefficient range queries in normal B-Trees. For this reason, B+-Trees are the most commonly used index structure in database systems @elmasri2016[p. 622]. 
The main difference between B-Trees and B+-Trees is that in B+-Trees, all data pointers are stored in the leafe nodes, resulting that internal nodes only store keys and pointers to child nodes, but no record pointers to the actual data records @elmasri2016[p. 622].
The advantage now is that all leaf nodes are linked together in a linked list, allowing for efficient range queries by following the pointer to the next leaf node after finding the start key in the leaf node.

#figure(
  caption: [Full mapping of B+ Tree leaf nodes to physical disk blocks via Record Pointers ($"RP"$).],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let leaf-fill = blue.lighten(95%)
    let disk-fill = orange.lighten(95%)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    let next-pointer-style = (stroke: blue + 0.8pt, mark: (end: "stealth", fill: blue, scale: 0.5))
    let rp-style = (stroke: gray + 0.5pt, mark: (end: "circle", fill: gray, scale: 0.2), dash: "densely-dotted")
    
    let leaf-page(pos, name, k1, k2) = {
      group(name: name, {
        // Increased width from 2.4 to 3.2
        rect(pos, (rel: (3.2, -0.6)), fill: leaf-fill, name: "box")
        
        // Dividers adjusted for the new width
        line((rel: (1.2, 0), to: pos), (rel: (1.2, -0.6), to: pos)) 
        line((rel: (2.4, 0), to: pos), (rel: (2.4, -0.6), to: pos)) 
        
        // Content centered in the new wider slots
        content((rel: (0.6, -0.3), to: pos), text(size: 8pt)[#k1, $"RP"_#k1$])
        content((rel: (1.8, -0.3), to: pos), text(size: 8pt)[#k2, $"RP"_#k2$])
        content((rel: (2.8, -0.3), to: pos), text(size: 8pt, fill: blue)[$P_n$])
      })
    }

    // --- Helper function for Disk Block ---
    let disk-block(pos, name, label) = {
      rect(pos, (rel: (1.2, -0.5)), fill: disk-fill, name: name, radius: 0.1)
      content(name, text(size: 7pt)[#label Data])
    }

    // --- Level 1: Internal Node ---
    rect((-0.8, 0.5), (0.8, -0.1), ..node-style, name: "L1")
    content("L1", [*12 | 15*])
    
    // --- Level 2: Leaves (Spaced out more to accommodate width) ---
    leaf-page((-5.5, -1.5), "leaf1", "5", "10")
    leaf-page((-1.6, -1.5), "leaf2", "12", "14")
    leaf-page((2.3, -1.5), "leaf3", "15", "20")

    // --- Level 3: The Physical Disk Layer ---
    line((-6, -2.8), (6, -2.8), stroke: (paint: gray, thickness: 1pt, dash: "dashed"))
    content((5.2, -2.6), text(size: 8pt, fill: gray)[Physical Disk])

    disk-block((-5.0, -3.5), "d5", "Key 5")
    disk-block((-3.6, -3.5), "d10", "Key 10")
    disk-block((-1.8, -3.5), "d12", "Key 12")
    disk-block((-0.4, -3.5), "d14", "Key 14")
    disk-block((2.1, -3.5), "d15", "Key 15")
    disk-block((3.5, -3.5), "d20", "Key 20")

    // --- Connections: Tree to Leaf (Targeting center of wide leaf) ---
    line("L1.south", (rel: (1.6, 0), to: "leaf1.north"), ..edge-style)
    line("L1.south", (rel: (1.6, 0), to: "leaf2.north"), ..edge-style)
    line("L1.south", (rel: (1.6, 0), to: "leaf3.north"), ..edge-style)

    // --- Connections: Sequential P_next ---
    line((rel: (2.8, -0.3), to: "leaf1.box.north-west"), (rel: (0, -0.3), to: "leaf2.box.north-west"), ..next-pointer-style)
    line((rel: (2.8, -0.3), to: "leaf2.box.north-west"), (rel: (0, -0.3), to: "leaf3.box.north-west"), ..next-pointer-style)
    

    // --- Connections: RP to Disk ---
    line((rel: (0.6, -0.6), to: "leaf1.box.north-west"), "d5.north", ..rp-style)
    line((rel: (1.8, -0.6), to: "leaf1.box.north-west"), "d10.north", ..rp-style)
    line((rel: (0.6, -0.6), to: "leaf2.box.north-west"), "d12.north", ..rp-style)
    line((rel: (1.8, -0.6), to: "leaf2.box.north-west"), "d14.north", ..rp-style)
    line((rel: (0.6, -0.6), to: "leaf3.box.north-west"), "d15.north", ..rp-style)
    line((rel: (1.8, -0.6), to: "leaf3.box.north-west"), "d20.north", ..rp-style)
  })
) <b-plus-disk-mapping>

Todo: standard disk-based index structures such as the B-tree will effectively double the I/O cost of the
transaction to maintain an index such as this in real time, increasing the total system cost up to
fifty percent @lsm_original[p. 351].



=== LSM-Trees (Log-Structured Merge Trees)
Log-Structured Merge Trees (LSM-Trees) are a type of index structure designed for high write throughput @lsm_original[p. 351] and was originally proposed by O'Neil et al. in 1996 @lsm_original.

==== LSM-Tree Structure according to O'Neil et al.
Thhe fundamental concept of an LSM-Tree is based to batch writes together for index updates, meaning not immediately updating the index on disk for each write operation, but instead writing to an in-memory structure and periodically merging it with the on-disk index @lsm_original[p. 355]. 
This is done using a hierachy of components (also called trees):

- *$C_0$ Component:* This is the in-memory component where all new writes are initially stored which could use a 2-3 Tree or AVL Tree, since it doesnt neet to insist on disk page size constraints @lsm_original[p. 356]. 2-3 or AVL Trees are another type of balanced search tree, which will not be explained in detail here, but they also have a logarithmic time complexity @intro_algorithms [358], @intro_algorithms [502]. All new writes are first written to this component, since it is in-memory, it allows for very fast write operations. This implies two things: First, the data needs to be written to the disk ($C_1$ Component) at some point and second, the data is not savely stored in case of a crash @lsm_original[p. 355]. 
- *$C_1$ Component:* This is now the on-disk component and larger than $C_0$. The data from $C_0$ is periodically merged into $C_1$ in a way that maintains the sorted order of the keys. The $C_1$ component is similar to a B-Tree, but optimized for sequential writes and reads with completely full nodes @lsm_original[p. 355].
- *$C_k$ Components:* In practice, there can be multiple on-disk components ($C_k$ with $k in N$) which are periodically merged together in a similar way to maintain the sorted order and optimize for read performance @lsm_original[p. 355].

As mentioned, the $C_0$ Component is periodically merged into the $C_1$ Component, which O'Neil et al. call a "rolling merge" @lsm_original[p. 355]. The rough idea is to merge the $C_0$ and $C_1$ components together, by using a merge sort-like process, where we read the sorted keys from both components and write them into a new on-disk component $C_1$ while maintaining the sorted order. Since this is done in a sequential manner, there is no need for seek time and rotational latency of discs, which allows for very efficient write operations in comparison to B-Trees @lsm_original[p. 358]. 

*Searching* in a LSM-Tree now works by starting in the in memory $C_0$ component and if the key is not found there, we continue searching in the on-disk components $C_1, C_2, ...$ starting from the lowest $C_k$ component until the key is found or all components have been searched. 

==== Google's Bigtable and Modern LSM-Trees
TODO: Shortly describe memtable etc, which DBMS use LSM trees? 

#pagebreak()

