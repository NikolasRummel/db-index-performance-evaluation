// Package index defines the common interfaces for database index implementations.
package index

// Index is the common interface for all database index implementations.
// It provides methods for basic CRUD operations and range queries.
type Index interface {
	// Insert adds a key-value pair to the index. If the key already exists,
	// its value is updated.
	Insert(key int64, value []byte) error

	// Get retrieves the value associated with the given key.
	// Returns nil if the key is not found.
	Get(key int64) ([]byte, error)

	// Delete removes the entry for the given key from the index.
	Delete(key int64) error

	// Range returns an iterator for scanning over a range of key-value pairs
	// from start to end (inclusive).
	Range(start, end int64) (Iterator, error)

	// Close flushes any pending changes and releases resources associated with the index.
	Close() error
}

// Iterator allows scanning over a range of key-value pairs in the index.
type Iterator interface {
	// Next advances the iterator to the next key-value pair.
	// It returns false when the end of the range is reached or an error occurs.
	Next() bool

	// Key returns the key of the current key-value pair.
	Key() int64

	// Value returns the value of the current key-value pair.
	Value() []byte

	// Error returns the first error encountered by the iterator, if any.
	Error() error

	// Close releases resources associated with the iterator.
	Close() error
}
