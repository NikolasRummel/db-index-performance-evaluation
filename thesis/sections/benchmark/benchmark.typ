#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"

= Design of the Comparison <design>

== Requirements and Constraints


== Coding language and libraries
For this project, the Go programming language was choosen for the implementation of the index structures and the benchmark. Inspired by the C programming language, Go is a statically typed, compiled language that however also provides high-level features like garbage collection and built-in support for concurrency @golang[preface p. xii] @godocs. Go was created by Google since they were dealing more and more with complex software systems @golang[preface p. xiiii] and now is widely used in the industry #footnote[https://survey.stackoverflow.co/2025/technology#most-popular-technologies-language]. 
With Go being a modern language, it provides a good balance between performance and ease of development, which makes it a good choice for implementing the index structures and the benchmark. Languares like C++ and Rust may be more performant but are more complex to work with, which is why Go was choosen. Additionally, Go has a huge standard library and a large ecosystem of third-party libraries that can be used to facilitate the implementation @godocs. There are also some bigger dbms like CockroachDB that are implemented in Go, which shows that it is a suitable language for database development @cockroachdb. 

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


