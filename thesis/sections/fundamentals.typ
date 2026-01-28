#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"

#pagebreak()

= Fundamentals

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


=== Search Tree-Based Index Structures
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
  caption: [Unbalanced search tree of degree $p=2$ with directed edges],
  cetz.canvas({
    import cetz.draw: *

    let node-style = (fill: white, stroke: 1pt, radius: 0.35)
    let edge-style = (mark: (end: "stealth", fill: black, scale: 0.5))
    
    circle((0, 0), ..node-style, name: "n10")
    content("n10", [*10*])
    
    circle((-1.5, -1), ..node-style, name: "n5")
    content("n5", [*5*])
    line("n10", "n5", ..edge-style)
    
    circle((1.5, -1), ..node-style, name: "n20")
    content("n20", [*20*])
    line("n10", "n20", ..edge-style)
    
    circle((0.8, -2), ..node-style, name: "n15")
    content("n15", [*15*])
    line("n20", "n15", ..edge-style)
    
    circle((2.2, -2), ..node-style, name: "n25")
    content("n25", [*25*])
    line("n20", "n25", ..edge-style)
    
    circle((2.8, -3), ..node-style, name: "n30")
    content("n30", [*30*])
    line("n25", "n30", ..edge-style)
    
    circle((3.4, -4), ..node-style, name: "n35")
    content("n35", [*35*])
    line("n30", "n35", ..edge-style)
  })
) <unbalanced-tree>

In this example, the search tree has a degree of $p=2$, meaning that each node can have at most one key and two child pointers. We see that the keys are organized in such a way that for each node, all keys in the left subtree are less than the key in the node, and all keys in the right subtree are greater.

Now, to search for a specific key in the tree, for instance 35, we start at the root node (10) and follow the pointers if our search key is less than or greater than the key in the node. In this case, since 35 is greater than 10, we follow the right pointer to node 20. We repeat this process until we either find the key or reach a leaf node.

==== Performance Considerations

In our case from @unbalanced-tree, the search tree is unbalanced, meaning that not all paths from the root node to all leafes have the same length @dbsystems_complete[p. 634]. Therefore, we can see that we almost searched all nodes to find the key 35. In the worst case, the tree could basically just be a linked list, where a search would result in $O(n)$ time complexity. With this there would not be any advantage of using a search tree over kust scanning the data. In order to avoid this problem, it makes sence to use a balanced search tree.

=== B-Trees
One of the most common balanced search trees used in database systems is the B-Tree @elmasri2016 [p. 620].


=== B+-Trees

//

=== LSM-Trees (Log-Structured Merge Trees)


#pagebreak()

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