#import "@preview/clean-dhbw:0.3.1": *
#import "@preview/cetz:0.4.2"

= Design of the Comparison <design>


== Implementation Specifications

=== Unified Interface

All implementations satisfy the `Index` interface contract (Go):

type Index interface {
    Insert(key int64, value []byte) error
    Get(key int64) ([]byte, error)
    Delete(key int64) error
    Range(start, end int64) (Iterator, error)
    SaveTo(path string) error
    LoadFrom(path string) error
    Close() error
}

type Iterator interface {
    Next() bool
    Key() int64
    Value() []byte
    Close() error
}
