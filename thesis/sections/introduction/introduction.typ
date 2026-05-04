#import "@preview/clean-dhbw:0.4.0": gls

= Introduction

== Motivation
Applications are getting more and more data intensive. Social media platforms have billions of users generating vast amounts of data daily, companies store huge amounts of data in database systems to analyze customer behavior or sales trends and with the rise of #gls("AI"), more and more data is being generated and processed to train machine learning models.
Depending on the use case, this data needs to be processed in different ways. For instance, in an #gls("OLTP") system, data needs to be inserted, updated and queried very fast to provide a good user experience. In contrast, in an #gls("OLAP") system, large amounts of data are analyzed to gain insights and generate reports for instance.

Traditionally, databases have used B+-Trees as index structures to optimize data access. However, with increasing data sizes and changing workloads, new index structures like LSM-Trees have been developed to address the challenges of write-intensive workloads and different variations of B-Trees are being used to optimize read performance.

The choice of the right index structure is crucial for the performance of a database system. Different index structures have different strengths and weaknesses, and the optimal choice depends on the specific workload and use case. Therefore, #gls("DBMS") developers must select the appropriate index for their specific software system, while software engineers need to understand the underlying principles of their chosen #gls("DBMS") to select the right platform and effectively optimize their application design for performance. 

== Objectives and Research Questions <research_questions>
To facilitate an informed decision on which index structure to use, this project aims to explore and compare the performance of different index structures. This will be done by implementing a benchmark that evaluates the performance of B-Trees, B+-Trees and LSM-Trees under different workloads and data sizes. For the implementation, a B-Tree and a B+-Tree will be implemented from scratch, while for the LSM-Tree, an already existing implementation will be used in order to save time and focus on the evaluation. 

The goal is to analyze their performance in various aspects like insertion speed or query speed for different workloads. While comparing the structures, the goal is also to answer the following research questions:

RQ1: How do B-Trees, B+-Trees, and LSM-Trees compare in terms of query speed?

RQ2: How significant is the performance gap between B-Trees and B+-Trees during range queries?

RQ3: Which index structure should you choose for a write- or read-heavy workload, considering factors like read/write ratio and data size?

== Structure of the Thesis
The thesis is structured in 5 main chapters. At first, an overview of #gls("DBMS") will be given, especially on how the storage management works. Secondly, in @index, the fundamentals of database index structures will be explained. In @design the actual implementation of index structures and the benchmark will be described. @evaluation will present the evaluation results and analyze them in detail and finally, in @conclusion, a summary of the findings will be given and an outlook on potential future work will be presented in @outlook.