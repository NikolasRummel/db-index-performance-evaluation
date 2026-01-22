= Introduction

== Motivation
Applications are getting more and more data intensive. Therefore, we need to optimize handling with that data.
TODO: Use cases with a lot of data.

Therefore, we need stuctures to process data retrievieval and manipulation efficiently.


== Problem Statement and Relevance
With increasing data sizes, the efficiency of data operations becomes crucial. Index structures play a vital role in optimizing these operations. This thesis aims to compare different index structures to determine their performance characteristics under various workloads.

== Objectives and Research Questions
The objective of this work is to compare different index structures from the most common dbms, so especially B tree variations and LSM trees. 
The goal is to analyze their performance in various aspects like insertion speed, query speed or memory usage. For this, the goal is also to answer the following research questions:

RQ1: How do B-Trees, B+-Trees, and LSM-Trees compare in terms of insertion speed under varying workloads?

RQ2: How significant is the performance gap between B-Trees and B+-Trees during range queries, and how does "Leaf Node Chaining" impact cache locality?

RQ3: Which index structure should you choose for a OLAP vs OLTP workload, considering factors like read/write ratio and data size?

RQ4: In a "No Index" scenario, at what dataset size does the overhead of maintaining an index structure become more efficient than a simple sequential scan?

== Structure of the Thesis