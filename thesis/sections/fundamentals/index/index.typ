#import "@preview/clean-dhbw:0.4.0": gls
#import "@preview/cetz:0.4.2"


= Index Structures in DBMS <index>

== Motivation for Database Index Structures
When the stored data in a #gls("DBMS") grows, it gets more and more important to efficiently queiry the data. 

To explain the basic concept, imagine a szenario of a user searching for movies and series of a specific actor at Netflix. With approx 8000 TODO: better example with more data titles available , a scan though all titles to find the one maching the search query would take a long time. 

To explain the basic concept, imagine a scenario where a user searches for a specific profile on a social media platform like Facebook. With approximately 3.07 billion users @statista_socialnetworks, a simple scan through all user records to find a single person would be practically impossible in real-time.

```sql
SELECT * FROM users WHERE username = 'Nikolas Rummel';
```

With a simple scan of the data, this query would have a performance of O(n), because in the worst case, all usernames have to be checked.
To optimize this search prozess, a datastructure saving the usernames in a smart way would benefit and reduce the amount of scans by a lot. This can be archieved by using an index structure, which allows to find the relevant data much faster.

An index is a data stucture allowing to quickly locate the data we are looking for, without having to scan all the data. We can define the index on a specific attribute A, for instance on our example on the username attribute @dbsystems_complete[P.~350]. This index will then speed up queries where A should match a specific value, like in our example above (A='Nikolas Rummel'). A very fast index would be a Hashmap, which could allow to find the title in O(1) time. However, which data structure is used for the index however depends on the use case and workload, which will be discussed in the following chapters.

== Types of Database Index Structures

As mentoned, there are different data structures which can be used as index structures in a database system. In the following, two of the most common index structures, B-Trees and LSM-Trees, will be explained in more detail.

=== Search Tree-Based Index Structures <search-trees>
#let p = $p$
In order to understand the most common data structure for DBMS, the B+-Tree @elmasri2016 [p. 618], we will start with standard search trees. 



A search tree is a type of data structure used to organize and manage data in a way that allows for efficient searching. It is defined with an order #p so that each #gls("Node") contains at most $p - 1$ keys and $p$ pointers in the sequence $<P_1, K_1, P_2, K_2, ..., P_(q-1), K_(q-1), P_q>$, where $q <= p$. Each $K_i$ is a key which we are searching for in our tree data structure, whereas each $P_i$ represents the pointers to child #gls("Node", plural: true) (including NULL pointers) @elmasri2016 [p. 618]. In other words, a #gls("Node") consists of values and pointers to other nodes. Each #gls("Node") have one more pointer than values since the pointers represent the subtrees between the values.

In order to be a search tree, this tree must satisfy the following constraints  @elmasri2016 [p. 618]:

1. In each node, the $K_i$ values are ordered such that $K_1 < K_2 < dots < K_(p-1)$ holds.
2. For all values $X$ in the subtree rooted at #gls("Node") $P_i$, the following conditions hold:
  - For $1 < i < q$: $K_(i-1) < X < K_i$
  - For $i = 1$: $X < K_1$
  - For $i = q$: $K_(q-1) < X$

In summary, this means that for each #gls("Node") in the tree, all keys in the left subtree are less than the key in the node, and all keys in the right subtree are greater. 


A search tree can then be visualized as follows:
 
#figure(
  caption: [Unbalanced tree structure of order $p=3$.],
  cetz.canvas(length: 0.8cm, {
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let ptr-style = (fill: rgb("#e1f5fe"), stroke: 0.5pt)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    
    let btree-node(name, pos, keys) = {
      let n = keys.len()
      let ptr-w = 0.7
      let key-w = 1.1
      let h = 0.7
      let w = (ptr-w * (n + 1)) + (key-w * n)
      
      group(name: name, {
        rect((pos.at(0) - w/2, pos.at(1) + h/2), (pos.at(0) + w/2, pos.at(1) - h/2), ..node-style)
        
        for i in range(n) {
          let x = (pos.at(0) - w/2) + (i * (ptr-w + key-w))
          rect((x, pos.at(1) + h/2), (x + ptr-w, pos.at(1) - h/2), ..ptr-style, name: "p" + str(i))
          content("p" + str(i), [$P_#i$], size: 8pt)
          content((x + ptr-w + key-w/2, pos.at(1)), [*#keys.at(i)*])
          line((x + ptr-w + key-w, pos.at(1) + h/2), (x + ptr-w + key-w, pos.at(1) - h/2))
        }
        
        let last-x = (pos.at(0) + w/2 - ptr-w)
        rect((last-x, pos.at(1) + h/2), (last-x + ptr-w, pos.at(1) - h/2), ..ptr-style, name: "p" + str(n))
        content("p" + str(n), [$P_#n$], size: 8pt)
      })
    }

    // Root
    btree-node("n10", (0, 0), ("10", "20"))
    
    // Level 1 — tighter horizontal spread
    btree-node("n5",  (-4.5, -3.0), ("5",))
    btree-node("n15", (0,    -3.0), ("12", "15"))
    btree-node("n25", (4.5,  -3.0), ("25", "30"))
    
    // Level 2
    btree-node("n35", (7.0, -6.0), ("35", "40"))
    
    // Level 3
    btree-node("n45", (9.5, -9.0), ("45", "50"))

    // Edges
    line("n10.p0.south", "n5.north",  ..edge-style)
    line("n10.p1.south", "n15.north", ..edge-style)
    line("n10.p2.south", "n25.north", ..edge-style)
    
    line("n25.p2.south", "n35.north", ..edge-style)
    line("n35.p2.south", "n45.north", ..edge-style)
  })
) <unbalanced-tree>


In this example, the search tree has a degree of $p=3$, meaning that each #gls("Node") can have at most one key and two child pointers. We see that the keys are organized in such a way that for each node, all keys in the left subtree are less than the key in the node, and all keys in the right subtree are greater.

==== Performance Considerations

In our case from @unbalanced-tree, the search tree is unbalanced, meaning that not all paths from the root #gls("Node") to all leafes have the same length @dbsystems_complete[p. 634]. Therefore, we can see that we almost searched all #gls("Node", plural: true) to find the key 35. In the worst case, the tree could basically just be a linked list, where a search would result in $O(n)$ time complexity. With this there would not be any advantage of using a search tree over kust scanning the data. In order to avoid this problem, it makes sence to use a balanced search tree.

=== B-Trees <btree>
To ensure that a search tree stays balanced, we can use a B-Tree. They where first described by Bayer and McCreight in 1972 @btree_original and are widely used in database systems both releational and non-relational @kleppmann[p. 80].
B-Trees are search trees with some additional constraints to ensure that the tree remains balanced @elmasri2016[p. 619], thus maintaining a $O(log n)$ time complexity for search, insert, and delete operations. Another reason in favor of B-Trees is that the #gls("Node") size can be fixed to a database #gls("Page") size. This alignment ensures that adding or removing a #gls("Node") corresponds exactly to the allocation or deallocation of a single database #gls("Page"), allowing the buffer manager to efficiently manage the pages. 

In addition, B-Trees were designed because the complete index structure does not fit in memory, so the tree is stored on disk @btree_original[p. 173]. This means that only a part of the tree is in memory at any given time, and the rest is stored on disk. To manage this, each B-Tree #gls("Node") contains a list of keys, pointers to child #gls("Node", plural: true) and record pointers to the actual data records on disk. 

#figure(
  caption: [Balenced version of the previous tree structure in @unbalanced-tree],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let ptr-style = (fill: rgb("#e1f5fe"), stroke: 0.5pt) 
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    
    let btree-node(name, pos, keys) = {
      let n = keys.len()
      let ptr-w = 0.4
      let key-w = 0.7
      let rp-w  = 0.5
      let w = (ptr-w * (n + 1)) + ((key-w + rp-w) * n)
      let h = 0.9
      
      group(name: name, {
        rect((pos.at(0) - w/2, pos.at(1) + h/2), (pos.at(0) + w/2, pos.at(1) - h/2), ..node-style)
        
        for i in range(n) {
          let x = (pos.at(0) - w/2) + (i * (ptr-w + key-w + rp-w))
          
          rect((x, pos.at(1) + h/2), (x + ptr-w, pos.at(1) - h/2), ..ptr-style, name: "p" + str(i))
          content("p" + str(i), text(size: 8pt)[$P_#i$])
          
          content((x + ptr-w + key-w/2, pos.at(1)), text(size: 9pt)[*#keys.at(i)*])
          
          line((x + ptr-w + key-w, pos.at(1) + h/2), (x + ptr-w + key-w, pos.at(1) - h/2))

          rect((x + ptr-w + key-w, pos.at(1) + h/2), (x + ptr-w + key-w + rp-w, pos.at(1) - h/2),
               fill: orange.lighten(80%), stroke: 0.5pt)
          content((x + ptr-w + key-w + rp-w/2, pos.at(1)), text(size: 5.5pt)[$"RP"_#i$])

          line((x + ptr-w + key-w + rp-w, pos.at(1) + h/2), (x + ptr-w + key-w + rp-w, pos.at(1) - h/2))
        }
        
        let last-p-x = (pos.at(0) + w/2 - ptr-w)
        rect((last-p-x, pos.at(1) + h/2), (last-p-x + ptr-w, pos.at(1) - h/2), ..ptr-style, name: "p" + str(n))
        content("p" + str(n), text(size: 8pt)[$P_#n$])
      })
    }

    // --- DRAW NODES ---
    
    btree-node("root", (0, 0), ("25",))
    btree-node("L1", (-3.5, -2.5), ("12",))
    btree-node("R1", (3.5, -2.5), ("35", "45"))
    
    let leaf-y = -5.0
    btree-node("leaf1", (-5.5, leaf-y), ("5", "10"))
    btree-node("leaf2", (-1.7, leaf-y), ("15", "20"))
    btree-node("leaf3", (1.9,  leaf-y), ("30",))
    btree-node("leaf4", (4.05,  leaf-y), ("40",))
    btree-node("leaf5", (6.2,  leaf-y), ("50",))

    // --- DRAW EDGES ---
    
    line("root.p0.south", "L1.north", ..edge-style)
    line("root.p1.south", "R1.north", ..edge-style)
    
    line("L1.p0.south", "leaf1.north", ..edge-style)
    line("L1.p1.south", "leaf2.north", ..edge-style)
    
    line("R1.p0.south", "leaf3.north", ..edge-style)
    line("R1.p1.south", "leaf4.north", ..edge-style)
    line("R1.p2.south", "leaf5.north", ..edge-style)
  })
) <balanced-tree>

The constraints for a B-Tree of order $p$ are as follows @elmasri2016[p. 619] @dbsystems_complete[pp. 634-635]:
1. Like in normal search trees, we have a alternating sequence of keys and pointers in each node. However, now a B-Tree stores the values on the disc, so each entry need also a record pointer to the actual data record. This result in a #gls("Node") structure of $<P_1, (K_1, "RP"_1), P_2, (K_2, "RP"_2), ..., P_(q-1), (K_(q-1), "RP"_(q-1)), P_q>$, where each $"RP"_i$ is the record pointer to the actual data record and $q <= p$.
2. Like in @search-trees, the keys in each #gls("Node") are ordered such that $K_1 < K_2 < dots < K_(q-1)$ holds.
3. All search key values $X$ within a subtree pointed by $P_i$ are bounded by the keys of the parent node. This ensures that subtrees contain only values from its parents key space. More formally, for all values $X$ in the subtree rooted at #gls("Node") $P_i$, the following conditions hold @elmasri2016 [p. 619]:
  - For $1 < i < q$: $K_(i-1) < X < K_i$
  - For $i = 1$: $X < K_i$
  - For $i = q$: $K_(i-1) < X$
4. To prevent a #gls("Node") from degenerating into a long, linear search structure and to ensure the tree grows vertically, each #gls("Node") is capped at $p$ tree pointers, where $p$ is typically chosen so that the entire #gls("Node") fits exactly within the size of a single database #gls("Page") to optimize buffer management as mentioned in the beginning of this section.
5. All #gls("Internal Node", plural: true) (except the root and leaves) have at least $ceil(p/2)$ tree pointers to ensure some kind of density to avoid wasting space. However, the root #gls("Node") has at least two tree pointers if it is not a #gls("Leaf Node") meaning its the only #gls("Node") in the tree.
6. All #gls("Leaf Node", plural: true) tree pointers $P_i$ are NULL and appear in the same level. This ensures that the tree is balanced and we get a guaranteed read performance of $O(h)$ with $h=log_p n$, where $n$ is the number of keys in the tree @intro_algorithms [p. 505]

==== Lookup 
A lookup operation in a B-Tree is similar to a normal search tree like in @unbalanced-tree @kleppmann[p. 80]. We start at the root #gls("Node") and compare the search key with the keys in the node. If we find a match, we return the corresponding record pointer to the disc and read from there the according value. If not, we follow the appropriate pointer to the child #gls("Node") based on the key comparison. We repeat this process until we either find the key or reach a NULL pointer in a #gls("Leaf Node"), meaning the search key is not present in the tree.

==== Insertion 
Insertion might seem trivial at first, if there is some space in a node, insert the new key in the correct position to maintain the order. However as described Garcia-Molina et al. @dbsystems_complete [pp. 640-641], if the #gls("Node") is full, we need to split the #gls("Node") into two #gls("Node", plural: true) and promote the middle key to the parent #gls("Node") to maintain the B-Tree properties. This process may propagate up to the root node, potentially increasing the height of the tree @btree_original[p. 178].

==== Deletion 
Again, deletion is most likely to be more complex than just removing the key from the node. First, it performes a lookup to find the key to be deleted @dbsystems_complete[p. 642] @btree_original[p. 190]. If the key is found in a #gls("Leaf Node"), it can be removed directly. However, if the key is in an #gls("Internal Node"), we again need to maintain the B-Tree properties described in @dbsystems_complete[p. 643] @btree_original[pp 180-182].

==== Range Query
Rangies Querys are a common operation in database systems, where we want to retrieve all keys within a specific range, for instance all titles between 'A' and 'D' @dbsystems_complete[p. 639]. For this example, we would do a lookup for the start key 'A' and then traverse the #gls("Leaf Node", plural: true) sequentially until we reach the end key 'D' with depth first in order walk. However, this will traverse all #gls("Node", plural: true) in the range, which can be inefficient if the range is large. B+-Trees address this issue with #gls("Leaf Node") chaining @, which will be explained in the next section.

=== B+-Trees <b-plus>
B+-Trees are a variant of B-Trees, fixing the problem of inefficient range queries in normal B-Trees. For this reason, B+-Trees are the most commonly used index structure in database systems @elmasri2016[p. 622]. 
The main difference between B-Trees and B+-Trees is that in B+-Trees, all data pointers are stored in the leafe nodes, resulting that #gls("Internal Node", plural: true) only store keys and pointers to child nodes, but no record pointers to the actual data records @elmasri2016[p. 622].
The advantage now is that all #gls("Leaf Node", plural: true) are linked together in a linked list, allowing for efficient range queries by following the pointer to the next #gls("Leaf Node") after finding the start key in the #gls("Leaf Node").

#figure(
  caption: [Full mapping of B+ Tree Leaf Nodes to physical disk blocks via Record Pointers ($"RP"$).],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt)
    let leaf-fill = blue.lighten(95%)
    let ptr-fill = rgb("#e1f5fe")
    let disk-fill = orange.lighten(95%)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    let next-pointer-style = (stroke: blue + 0.8pt, mark: (end: "stealth", fill: blue, scale: 0.5))
    let rp-style = (stroke: gray + 0.5pt, mark: (end: "circle", fill: gray, scale: 0.2), dash: "densely-dotted")

    let ptr-w = 0.65  // wider to fit RP_i label
    let key-w = 0.9
    let pn-w  = 0.6
    let h     = 0.6
    let total-w = 2 * ptr-w + 2 * key-w + pn-w


    let leaf-page(pos, name, k1, k2) = {
      let x = pos.at(0)
      let y = pos.at(1)
      group(name: name, {
        rect((x, y), (x + total-w, y - h), fill: leaf-fill, name: "box")
        rect((x, y), (x + ptr-w, y - h), fill: orange.lighten(80%), name: "p0")
        content("p0", text(size: 7pt)[$"RP"_0$])
        content((x + ptr-w + key-w/2, y - h/2), text(size: 8pt)[#k1])
        line((x + ptr-w + key-w, y), (x + ptr-w + key-w, y - h))
        let p1x = x + ptr-w + key-w
        rect((p1x, y), (p1x + ptr-w, y - h), fill: orange.lighten(80%), name: "p1")
        content("p1", text(size: 7pt)[$"RP"_1$])
        content((p1x + ptr-w + key-w/2, y - h/2), text(size: 8pt)[#k2])
        let pnx = x + total-w - pn-w
        line((pnx, y), (pnx, y - h))
        rect((pnx, y), (pnx + pn-w, y - h), fill: ptr-fill, name: "pn")
        content("pn", text(size: 7pt, fill: blue)[$P_n$])
      })
    }

    let disk-block(pos, name, label) = {
      rect(pos, (rel: (1.6, -0.5)), fill: disk-fill, name: name, radius: 0.1)
      content(name, text(size: 7pt)[#label Data])
    }

    // --- Root #gls("Node") with pointer slots ---
    let root-ptr-w = 0.7
    let root-key-w = 1.1
    let root-h = 0.7
    let root-keys = ("12", "15")
    let rn = root-keys.len()
    let root-w = (root-ptr-w * (rn + 1)) + (root-key-w * rn)
    let rx = -root-w / 2
    let ry = 0.5

    group(name: "root", {
      rect((rx, ry), (rx + root-w, ry - root-h), ..node-style)
      for i in range(rn) {
        let x = rx + (i * (root-ptr-w + root-key-w))
        rect((x, ry), (x + root-ptr-w, ry - root-h), fill: ptr-fill, name: "p" + str(i))
        content("p" + str(i), text(size: 8pt)[$P_#i$])
        content((x + root-ptr-w + root-key-w/2, ry - root-h/2), [*#root-keys.at(i)*])
        line((x + root-ptr-w + root-key-w, ry), (x + root-ptr-w + root-key-w, ry - root-h))
      }
      let last-x = rx + root-w - root-ptr-w
      rect((last-x, ry), (last-x + root-ptr-w, ry - root-h), fill: ptr-fill, name: "p" + str(rn))
      content("p" + str(rn), text(size: 8pt)[$P_#rn$])
    })

    // --- Level 2: Leaves ---
    leaf-page((-5.9, -1.5), "leaf1", "5", "10")
    leaf-page((-1.85, -1.5), "leaf2", "12", "14")
    leaf-page((2.2,  -1.5), "leaf3", "15", "20")

    // --- Physical Disk Layer ---
    line((-6.5, -2.8), (6.8, -2.8), stroke: (paint: gray, thickness: 1pt, dash: "dashed"))
    content((5.8, -2.6), text(size: 8pt, fill: gray)[Physical Disk])

    disk-block((-5.6, -3.5), "d5",  "Key 5")
    disk-block((-3.8, -3.5), "d10", "Key 10")
    disk-block((-2.0, -3.5), "d12", "Key 12")
    disk-block((-0.2, -3.5), "d14", "Key 14")
    disk-block((2.0,  -3.5), "d15", "Key 15")
    disk-block((3.8,  -3.5), "d20", "Key 20")

    // --- Connections: Root pointer slots to Leaves ---
    line("root.p0", (rel: (1.75, 0), to: "leaf1.box.north-west"), ..edge-style)
    line("root.p1", (rel: (1.75, 0), to: "leaf2.box.north-west"), ..edge-style)
    line("root.p2", (rel: (1.75, 0), to: "leaf3.box.north-west"), ..edge-style)

    // --- Connections: Sequential P_next ---
    line("leaf1.pn", (rel: (0, -0.3), to: "leaf2.box.north-west"), ..next-pointer-style)
    line("leaf2.pn", (rel: (0, -0.3), to: "leaf3.box.north-west"), ..next-pointer-style)

    // --- Connections: RP to Disk ---
    line("leaf1.p0", "d5.north",  ..rp-style)
    line("leaf1.p1", "d10.north", ..rp-style)
    line("leaf2.p0", "d12.north", ..rp-style)
    line("leaf2.p1", "d14.north", ..rp-style)
    line("leaf3.p0", "d15.north", ..rp-style)
    line("leaf3.p1", "d20.north", ..rp-style)
  })
) <b-plus-disk-mapping>

Now, for a lookup, we follow the same logic like in a normal B-Tree, but will wend in a leaf page in order to get the reccord pointer to the actual data record on the disk. For a range query, we can now follow the pointer to the next #gls("Leaf Node") after finding the start key in the #gls("Leaf Node"), which allows for efficient range queries by traversing the linked list of #gls("Leaf Node", plural: true). 


==== Drawbacks of B-Trees <drawbacks-btree>
However, the B+-Tree (and B-Tree as well) still is not a perfect solution for every possible scenario and they have some drawbacks. First, they are not optimized for write-heavy workloads, since each write operation requires multiple disk I/O operations to maintain the index structure on disk @lsm_original[p. 351]. This will effectively double the I/O cost of the
transaction to maintain an index such as this in real time, increasing the total system cost up to fifty percent @lsm_original[p. 351]. Secondly, after a page split a B-Tree, some space in those pages is wasted, which leads to fragmentation @kleppmann[p. 84]. In addition, those B-Trees are not crash-safe since they update in place, meaning that in case of a crash while a merge or split is happening, the tree could be left in an inconsistent state @lsm_original[p. 351]. To mitigate this problem, a #gls("WAL") can be used, which would lead do a lot of #gls("writeamplification"), since we would have to write the log entry and then write the actual data to the disk, which would double the write cost. @kleppmann[p. 82]


=== LSM-Trees (Log-Structured Merge Trees)
Log-Structured Merge Trees (LSM-Trees) are a type of index structure designed for high write throughput @lsm_original[p. 351] and was originally proposed by O'Neil et al. in 1996 @lsm_original. The reason for the design of a new index structure was that the standard disk-based index structures such as the B-tree have some those drawbacks mentioned in @drawbacks-btree, which are especially problematic for write-heavy workloads. LSM-Trees are designed to optimize write performance by batching writes together and writing them sequentially to disk, which minimizes the number of disk I/O operations required for each write and thus significantly improves write performance @lsm_original[p. 351]. 

==== LSM-Tree Structure according to O'Neil et al. <lsm_oneil>
The fundamental concept of an LSM-Tree is based to batch writes together for index updates, meaning not immediately updating the index on disk for each write operation, but instead writing to an in-memory structure and periodically merging it with the on-disk index @lsm_original[p. 355]. 
This is done using a hierachy of components (also called trees):

- *$C_0$ Component:* This is the in-memory component where all new writes are initially stored which could use a 2-3 Tree or AVL Tree, since it doesnt neet to insist on disk page size constraints @lsm_original[p. 356]. 2-3 or AVL Trees are another type of balanced search tree, which will not be explained in detail here, but they also have a logarithmic time complexity @intro_algorithms [358], @intro_algorithms [502]. All new writes are first written to this component, since it is in-memory, it allows for very fast write operations. This implies two things: First, the data needs to be written to the disk ($C_1$ Component) at some point and second, the data is not savely stored in case of a crash @lsm_original[p. 355]. 
- *$C_1$ Component:* This is now the on-disk component and larger than $C_0$. The data from $C_0$ is periodically merged into $C_1$ in a way that maintains the sorted order of the keys. The $C_1$ component is similar to a B-Tree, but optimized for sequential writes and reads with completely full #gls("Node", plural: true) @lsm_original[p. 355].
- *$C_k$ Components:* In practice, there can be multiple on-disk components ($C_k$ with $k in N$) which are periodically merged together in a similar way to maintain the sorted order and optimize for read performance @lsm_original[p. 355].

*Insertion* in a LSM-Tree works by first writing the new key-value pair to the in-memory $C_0$ component. When the $C_0$ component becomes full, it is merged with the on-disk $C_1$ component, and the process repeats.

*Updating* of a key-value pair in a LSM-Tree is basically an insertion. However, the old value still exists in the LSM-Tree. This is because the new value is written to the $C_0$ component, while the old value still exists in the $C_1$ component. 

*Deletion* of a key-value pair in a LSM-Tree is also an insertion, but instead of writing the new value, we write a special tombstone value to the $C_0$ component. This tombstone value indicates that the key has been deleted, and when the $C_0$ component is merged with the $C_1$ component, the old value will be removed from the on-disk component.

A *Lookup* in a LSM-Tree now works by starting in the in memory $C_0$ component and if the key is not found there, we continue searching in the on-disk components $C_1, C_2, ...$ starting from the lowest $C_k$ component until the key is found, a tombstone appears or all components have been searched. This can lead to inefficient read performance, since we might have to search through multiple on-disk components, which is a major drawback of LSM-Trees. To mitigate this problem, LSM-Trees often use #gls("Bloom"), a data structure to quickly check if a key is likely to be present in an on-disk component before performing a more expensive search @kleppmann[p. 79].  

As mentioned, the $C_0$ Component is periodically merged into the $C_1$ Component, which O'Neil et al. call a "rolling merge" @lsm_original[p. 355]. The rough idea is to merge the $C_0$ and $C_1$ components together, by using a merge sort-like process, where we read the sorted keys from both components and write them into a new on-disk component $C_1$ while maintaining the sorted order. Since this is done in a sequential manner, there is no need for seek time and rotational latency of discs, which allows for very efficient write operations in comparison to B-Trees @lsm_original[p. 358]. This process is then repeated for the other on-disk components $C_k$ with $k in N$ as well, where we merge the smaller, higher-level component $C_n$ with the larger, lower-level $C_(n+1)$ to produce a new, optimized $C_(n+1)$ component @lsm_original[p. 355].

An example of this process is shown in the following figure @lsm-rolling-merge2. Imagine a bank account database where we have a $C_n$ component with recent updates and deletions, and an old $C_(n+1)$ component with existing entries. During the merge, the updated values from $C_n$ will replace their older counterparts in $C_(n+1)$, while tombstone markers indicating deletions will lead to the removal of those entries in the new $C_(n+1)$ component. This process ensures that the new on-disk component reflects the most up-to-date state of the data while maintaining efficient write performance @kleppmann[p. 79].

#figure(
  caption: [Example of rolling merge process. The smaller upper component carries both updated values and tombstone markers (#smallcaps[del]). During the merge new values replace their older counterparts in $C_(n+1)$, while tombstones lead to deletion meaning they are not written to the new $C_(n+1)$. Adapted from @kleppmann[Fig. 3.3, p. 74]],
  cetz.canvas({
    import cetz.draw: *

    let node-style  = (fill: white, stroke: 1pt)
    let leaf-fill   = blue.lighten(95%)
    let disk-fill   = orange.lighten(95%)
    let result-fill = green.lighten(95%)
    let del-color   = red.darken(10%)
    let edge-style  = (mark: (end: "stealth", fill: black, scale: 0.5))
    let s           = 0.5pt + gray

    // ── Cn (upper / smaller) — two rows ──
    rect((-6.5, 6.2), (-0.5, 3.5), fill: leaf-fill, stroke: 1pt, radius: 0.05)
    content((-3.5, 5.95), text(size: 9pt, weight: "bold")[ $C_n$])
    content((-3.5, 5.65), text(size: 7pt, fill: gray)[Sorted segment — recent writes])

    // row 1
    rect((-6.2, 5.35), (rel: (5.4, -0.75)), fill: white, stroke: s)
    line((-4.4, 5.35), (-4.4, 4.6), stroke: s)
    line((-2.6, 5.35), (-2.6, 4.6), stroke: s)
    content((-5.3, 5.08), text(size: 8pt)[Max: 1000])
    content((-5.3, 4.78), text(size: 7pt, fill: gray)[first write])
    content((-3.5, 5.08), text(size: 8pt)[Lisa: 2000])
    content((-3.5, 4.78), text(size: 7pt, fill: gray)[first write])
    content((-1.7, 5.08), text(size: 8pt)[Anna: 1500])
    content((-1.7, 4.78), text(size: 7pt, fill: gray)[first write])

    // row 2
    rect((-6.2, 4.5), (rel: (5.4, -0.75)), fill: white, stroke: s)
    line((-4.4, 4.5), (-4.4, 3.75), stroke: s)
    line((-2.6, 4.5), (-2.6, 3.75), stroke: s)
    content((-5.3, 4.23), text(size: 8pt)[Max: 1350])
    content((-5.3, 3.93), text(size: 7pt, fill: gray)[update])
    content((-3.5, 4.23), text(size: 8pt, fill: del-color)[Lisa: DEL])
    content((-3.5, 3.93), text(size: 7pt, fill: gray)[tombstone])
    content((-1.7, 4.23), text(size: 8pt)[Anna: 1900])
    content((-1.7, 3.93), text(size: 7pt, fill: gray)[update])

    // ── Old Cn+1 (lower / larger) — two rows ──
    rect((0.5, 6.2), (6.5, 3.5), fill: disk-fill, stroke: 1pt, radius: 0.05)
    content((3.5, 5.95), text(size: 9pt, weight: "bold")[ $C_(n+1)$])
    content((3.5, 5.65), text(size: 7pt, fill: gray)[Sorted segment — existing entries])

    // row 1
    rect((0.8, 5.35), (rel: (5.4, -0.75)), fill: white, stroke: s)
    line((2.6, 5.35), (2.6, 4.6), stroke: s)
    line((4.4, 5.35), (4.4, 4.6), stroke: s)
    content((1.7, 5.08), text(size: 8pt)[Max: 500])
    content((1.7, 4.78), text(size: 7pt, fill: gray)[original])
    content((3.5, 5.08), text(size: 8pt)[Lisa: 800])
    content((3.5, 4.78), text(size: 7pt, fill: gray)[original])
    content((5.3, 5.08), text(size: 8pt)[Anna: 600])
    content((5.3, 4.78), text(size: 7pt, fill: gray)[original])

    // row 2
    rect((0.8, 4.5), (rel: (5.4, -0.75)), fill: white, stroke: s)
    line((2.6, 4.5), (2.6, 3.75), stroke: s)
    line((4.4, 4.5), (4.4, 3.75), stroke: s)
    content((1.7, 4.23), text(size: 8pt)[Max: 1000])
    content((1.7, 3.93), text(size: 7pt, fill: gray)[update])
    content((3.5, 4.23), text(size: 8pt)[Lisa: 2000])
    content((3.5, 3.93), text(size: 7pt, fill: gray)[update])
    content((5.3, 4.23), text(size: 8pt)[Anna: 1500])
    content((5.3, 3.93), text(size: 7pt, fill: gray)[update])

    // ── Arrows into merge #gls("Node") ──
    line((-3.5, 3.5), (-0.5, 1.85), ..edge-style)
    line((3.5, 3.5), (0.5, 1.85), ..edge-style)

    // ── Rolling merge circle ──
    circle((0, 1.0), radius: 0.85, fill: white, stroke: 1pt)
    content((0, 1), align(center, text(size: 8pt, weight: "bold")[Merge \ Sort \ Logic]))


    // ── Arrow out ──
    line((0, 0.15), (0, -0.8), ..edge-style)

    // ── New Cn+1 result ──
    rect((-6.5, -0.9), (6.5, -3.2), fill: result-fill, stroke: 1pt, radius: 0.05)
    content((0, -1.25), text(size: 9pt, weight: "bold")[New $C_(n+1)$ — merged ])

    // result entry grid
    rect((-6.2, -1.65), (rel: (12.4, -1.3)), fill: white, stroke: s)
    line((-2.07, -1.65), (-2.07, -2.95), stroke: s)
    line((2.07, -1.65), (2.07, -2.95), stroke: s)

    // Anna: kept
    content((-4.14, -2.1),  text(size: 8pt, weight: "bold")[Anna: 1900])
    content((-4.14, -2.45), text(size: 7pt, fill: gray)[latest update kept])

    // Max: kept
    content((0, -2.1),  text(size: 8pt, weight: "bold")[Max: 1350])
    content((0, -2.45), text(size: 7pt, fill: gray)[latest update kept])

    // Lisa: deleted
    content((4.14, -2.1),  text(size: 8pt, fill: gray)[Lisa])
    content((4.14, -2.45), text(size: 7pt, fill: gray)[tombstone — not written])
    line((2.19, -1.7), (6.09, -2.9), stroke: 0.8pt + del-color)
    line((6.09, -1.7), (2.19, -2.9), stroke: 0.8pt + del-color)
  })
) <lsm-rolling-merge2>


==== LSM-Tree Structure in Practice
The main idea of O'Neil et al. is still the same, but since Google published `Bigtable`, the common term for the $C_0$ component is #gls("Memtable"), while the on-disk components collections of so called #gls("SSTable", plural: true) @lsm_survey[p. 2]@kleppmann[p. 78].  #gls("SSTable", plural: true) are #gls("Memtable", plural: true) which are written sequentially in a sorted order, making the LSM-Tree fast in flushing @lsm_survey[p. 2]. In addition, next to a #gls("SSTable"), a small index is maintained, that is used to provide lookup for the particular #gls("SSTable") @lsm_survey[p. 2]. Also, all LSM-Trees use #gls("Bloom") to optimize read performance, which is suggested in the original paper by O'Neil et al. @lsm_original[p. 381]. To ensure data durability, the #gls("Memtable") is often backed by a #gls("WAL"), which is written to disk before the #gls("Memtable") is updated. This way, in case of a crash, the system can recover the data from the log and rebuild the #gls("Memtable") @lsm_survey[p. 2].

A complete architecture of an LSM-Tree know looks like the following in @lsm_fig.

#figure(
  image("../../../assets/lsm-tree.png", width: 100%),
  caption: [Complete architecture of an LSM-Tree according to Supriya @lsm_survey[p. 2].], 
) <lsm_fig>

Here we see that those on-disk components $C_1, C_2, \ldots, C_n$ are organized in levels. Each level has a size limit, and when the size limit is reached, the data is merged into the next level with the #gls("Compaction") process explained in @lsm-rolling-merge2. Usually, the size limit is around 10 times the size of the level before, which means that the higher levels are much larger than the lower levels @cockroachdb_storage_layer.


==== Drawbacks of LSM-Trees
While LSM-Tree withthe rolling merge process ensures efficient sequential writes, it comes with some drawbacks. First of all, since its a append-only structure, it leads to increased storage requirements due to the presence of multiple on-disk components and tombstone markers for deletions. Secondly, this leads to a high #gls("writeamplification") since data exists in multiple components and needs to be merged multiple times. Lastly, the merge process takes some ressources and therefore decrese the overall performance of the tree, which will be discussed in @evaluation.