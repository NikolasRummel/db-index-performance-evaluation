package index

type Iterator interface {
	Next() bool
	Key() int64
	Value() []byte
	Error() error
	Close() error
}
