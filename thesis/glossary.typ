#let glossary-entries = (
  (
    key: "Softwareschnittstelle",
    description: "Ein logischer Berührungspunkt in einem Softwaresystem: Sie ermöglicht und regelt den Austausch von Kommandos und Daten zwischen verschiedenen Prozessen und Komponenten.",
  ),
  (
    key: "Komponente",
    description: "Ein Architekturbaustein. Zusammengesetzte Komponenten bestehen aus weiteren Subkomponenten. Einfache Komponenten sind nicht weiter unterteilt.",
  ),
  (
    key: "API",
    short: "API",
    long: "Application Programming Interface",
  ),
  (
    key: "HTTP",
    short: "HTTP",
    long: "Hypertext Transfer Protocol",
  ),
  (
    key: "DBMS",
    short: "DBMS",
    long: "Database Management System",
  ),
  (
    key: "SQL",
    short: "SQL",
    long: "Structured Query Language",
  ),
  (
    key: "HDD",
    short: "HDD",
    long: "Hard Disk Drive",
  ),
  (
    key: "SSD",
    short: "SSD",
    long: "Solid State Drive",
  ),
  (
    key: "ACID",
    description: "Atomicity, Consistency, Isolation, Durability. A set of properties that guarantee reliable processing of database transactions in relational databases.",
  ),
  (
    key: "IPO",
    description: "Input-Process-Output Model. A model that describes the typical structure of a program or system which follows this 3-step process.",
  ),

  (
    key: "CRUD",
    description: "Create, Read, Update, Delete. A common acronym for the four basic operations of persistent storage.",
  ),
  (
    key: "Bloom",
    short: "Bloom Filter",
    description: "Bloom filters are used to quickly check in $O(1)$ if an element is not present in a set. They are a space-efficient probabilistic data structure to reduce the number of disk accesses needed to find a key in a memtable or SSTable of a LSM-Tree.",
  ),
  (
    key: "writeamplification",
    description: "Write amplification is a phenomenon where the anoubt of data written to the storage media is greater than the amount of data intended to be written. The goal is to minimize write amplification, as it can lead to increased wear on storage devices and unnecessary writes.",
  )

)
