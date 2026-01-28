package index

type Index interface {
	Insert(key int64, value []byte) error
	Get(key int64) ([]byte, error)
	Delete(key int64) error
	Range(start, end int64) (Iterator, error)

	SaveTo(path string) error
	LoadFrom(path string) error
	Close() error
}
