#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"

= Design of the Comparison <design>

== Requirements and Constraints

== Coding language and libraries

== Index implementations


#figure(
  caption: "Ein Stück Quellcode",
  sourcecode[```go
    type Index interface {
        Insert(key int64, value []byte) error
        Get(key int64) ([]byte, error)
        Delete(key int64) error
        Range(start, end int64) (Iterator, error)
        Close() error
    }

    type Iterator interface {
        Next() bool
        Key() int64
        Value() []byte
        Close() error
    }
```],
)

=== B-Tree Implementation

=== B+-Tree Implementation

=== LSM-Tree Implementation

== Benchmark Design
=== Workload Generation
==== Insertion Workload
==== Point Query Workload
==== Range Query Workload
==== ...


