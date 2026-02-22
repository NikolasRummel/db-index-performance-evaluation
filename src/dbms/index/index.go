package index

// Index is the common interface for all implementations.
type Index interface {
	Insert(key int64, value []byte) error
	Get(key int64) ([]byte, error)
	Delete(key int64) error
	Range(start, end int64) (Iterator, error)
	Close() error
}

// Iterator allows scanning over a range of key-value pairs.
type Iterator interface {
	Next() bool
	Key() int64
	Value() []byte
	Error() error
	Close() error
}
