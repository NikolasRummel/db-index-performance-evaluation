= Introduction

== Motivation
Applications are getting more and more data intensive. Social media plattforms have millions of users generating vast amounts of data daily, companys store huge amount of data in database systems to analyze customer behavior or sales trends.
Depending on the use case, this data needs to be processed in different ways. For instance, in an online transaction processing (OLTP) system, data needs to be inserted, updated and queried very fast to provide a good user experience. In contrast, in an online analytical processing (OLAP) system, large amounts of data are analyzed to gain insights and generate reports.

Traditionally, database have used B+-Trees as index structures to optimize data access. However, with the increasing data sizes and changing workloads, new index structures like Log-Structured Merge-Trees (LSM-Trees) have been developed to address the challenges of write-intensive workloads <> and more and more variations of B-Trees are being used to optimize read performance <>.

The choice of the right index structure is crucial for the performance of a database system. Different index structures have different strengths and weaknesses, and the optimal choice depends on the specific workload and use case. 

== Objectives and Research Questions
The objective of this work is to compare different index structures from the most common dbms, so especially B tree variations and LSM trees. 
The goal is to analyze their performance in various aspects like insertion speed, query speed or memory usage. For this, the goal is also to answer the following research questions:

RQ1: How do B-Trees, B+-Trees, and LSM-Trees compare in terms of insertion speed under varying workloads?

RQ2: How significant is the performance gap between B-Trees and B+-Trees during range queries, and how does "Leaf Node Chaining" impact cache locality?

RQ3: Which index structure should you choose for a OLAP vs OLTP workload, considering factors like read/write ratio and data size?

== Structure of the Thesis
The thesis is structured in 4 main chapters. First in chapter @fundamentals, the fundamentals of database index structures will be explained and some recent research on different B tree variations and LSM trees will be presented. In chapter @design, the implementation of index structures and the benchmark will be described. Chapter @evaluation will present the evaluation results and analyze them in detail. Finally, in chapter @conclusion, a summary of the findings will be given and an outlook on potential future work will be presented in chapter @outlook.