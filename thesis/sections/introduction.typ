#import "@preview/clean-dhbw:0.3.1": gls

= Introduction

== Motivation
Applications are getting more and more data intensive. Social media plattforms have millions of users generating vast amounts of data daily, companys store huge amount of data in database systems to analyze customer behavior or sales trends and with the rise of AI, more and more data is being generated and processed to train machine learning models.
Depending on the use case, this data needs to be processed in different ways. For instance, in an online transaction processing (OLTP) system, data needs to be inserted, updated and queried very fast to provide a good user experience. In contrast, in an online analytical processing (OLAP) system, large amounts of data are analyzed to gain insights and generate reports for instance.

Traditionally, database have used B+-Trees as index structures to optimize data access. However, with the increasing data sizes and changing workloads, new index structures like Log-Structured Merge-Trees (LSM-Trees) have been developed to address the challenges of write-intensive workloads <> and more and more variations of B-Trees are being used to optimize read performance <>.

The choice of the right index structure is crucial for the performance of a database system. Different index structures have different strengths and weaknesses, and the optimal choice depends on the specific workload and use case. Therefore, #gls("DBMS") developers must select the appropriate index for their specific software system, while software engineers need to understand the underlying principles of their chosen #gls("DBMS") to select the right platform and effectively optimize their application design for performance

== Objectives and Research Questions
In order to make a decision on which index structure to use, this project aims to explore and compare the performance of different index structures. This will be done by implementing a benchmark that evaluates the performance of B-Trees, B+-Trees and LSM-Trees under different workloads and data sizes. For the implementation, a B-Tree and a B+-Tree will be implemented from scratch, while for the LSM-Tree, an existing implementation will be used in order to save time and focus on the evaluation. As inspiration the rough design of the B-Tree and B+-Tree implementations will be based on SQLite concept of storing all data in one file. 

The goal is to analyze their performance in various aspects like insertion speed, query speed or memory usage. While comparing the structutes, the goal is also to answer the following research questions:

RQ1: How do B-Trees, B+-Trees, and LSM-Trees compare in terms of insertion speed under varying workloads?

RQ2: How significant is the performance gap between B-Trees and B+-Trees during range queries, and how does "Leaf Node Chaining" impact cache locality?

RQ3: Which index structure should you choose for a OLAP vs OLTP workload, considering factors like read/write ratio and data size?

== Structure of the Thesis
The thesis is structured in 5 main chapters. At first, some overview od #gls("DBMS") will be given, espacially on the storage management. Secondly, in chapter @index, the fundamentals of database index structures will be explained. In chapter @design, the actual implementation of index structures and the benchmark will be described. Chapter @evaluation will present the evaluation results and analyze them in detail and finally, in chapter @conclusion, a summary of the findings will be given and an outlook on potential future work will be presented in chapter @outlook.