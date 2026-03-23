#import "@preview/clean-dhbw:0.4.0": gls

= Overview DBMS <dbms>

#align(right)[
  #pad(left: 25%)[
    _ " A #gls("DBMS")  is a computerized system that enables users to create and maintain a database. The DBMS is a general-purpose software system that facilitates the processes of defining, constructing, manipulating, and sharing databases among various users and applications." _

    --- *R. Elmasri & S. B. Navathe* @elmasri2016[p. 6]
  ]
]

To understand index structures and their implementation in a #gls("DBMS"), one must first understand the basic components and architecture of a #gls("DBMS"). A #gls("DBMS") is not just a simple programm, but rather a complex system that consists of several components that work together to manage and manipulate data efficiently. While the definition by Elmasri and Navathe provides a idea on fumctional requirements of a #gls("DBMS"), its architecture shows the mechanisms that enable it to fulfill these requirements. The architectural components of a #gls("DBMS") can be broadly categorized into the following:

#figure(
  image("../../../assets/dbms.png", width: 100%),
  caption: [Architectural components of a DBMS according to Elmasri and Navathe @elmasri2016[p. 43].], 
) <dbms_fig>

== Query Processing and Data Definition 
Users and database administrators interact with the #gls("DBMS") though specific ddl and dml statements. The *DDL Compiler* processes data definition statements, which are used to define the database schema, including tables, indexes, and other database objects. For end users, the *Query Processor* is responsible for data manipulation and retrieval. 
For relational databases, the most common language for both database administrators and end users is #gls("SQL"). Both the processor and compiler then forward their parsed statements to the *Execution Engine*, which usually optimizes the execution plan and then executes the statements. 

== Component B 
Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua.
