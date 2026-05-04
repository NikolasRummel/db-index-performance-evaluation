#import "@preview/clean-dhbw:0.4.0": gls

= Overview DBMS <dbms>

#align(right)[
  #pad(left: 25%)[
    _ " A #gls("DBMS") is a computerized system that enables users to create and maintain a database. The #gls("DBMS") is a general-purpose software system that facilitates the processes of defining, constructing, manipulating, and sharing databases among various users and applications." _

    --- *R. Elmasri & S. B. Navathe* @elmasri2016[p. 6]
  ]
]

To understand index structures and their implementation in a #gls("DBMS"), one must first understand the basic components and architecture of a #gls("DBMS"). A #gls("DBMS") is not just a simple program, but rather a complex system that consists of several components that work together to manage and manipulate data efficiently. While the definition by Elmasri and Navathe provides a rough idea of the functional requirements in a #gls("DBMS"), its architecture consists of multiple components to fulfill those. In the following, a rough description of the main components shown in @dbms_fig will be given.

#figure(
  image("../../../assets/dbms.png", width: 100%),
  caption: [Architectural components of a DBMS according to Elmasri and Navathe @elmasri2016[p. 43].],
) <dbms_fig>

== Query Processing and Data Definition 
Users and database administrators interact with the #gls("DBMS") through specific #gls("DDL") and #gls("DML") statements. The `DDL Compiler` processes data definition statements, which are used to define the database schema, including tables, indexes, and other database objects. For end users, the `Query Processor` is responsible for data manipulation and retrieval. 
For relational databases, the most common language for both database administrators and end users is #gls("SQL"). Both the processor and compiler then forward their parsed statements to the `Query optimizer`, which improves the execution plan and forwards it for execution. 

== Execution of Queries and Transactions
After the query was optimized, the `Runtime Database Processor` executes the query by interacting with the `Storage Manager`, which will be discussed in @storage. In addition, it must communicate with the `System Catalog`, which contains metadata about the database, such as the structure of tables, indexes, and other database objects, to verify that requested tables exist. In order to ensure the #gls("ACID") properties, concurrency control and recovery management are also crucial components of a #gls("DBMS"). 
