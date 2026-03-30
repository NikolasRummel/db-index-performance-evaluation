// Package listindex implements a simple in-memory list-based index for testing.
package listindex

import (
	"errors"
	"slices"

	"github.com/btree-query-bench/bmark/dbms/index"
)

var _ index.Index = (*ListIndex)(nil)

// Data represents a key-value pair in the ListIndex.
type Data struct {
	Key int64
	Val []byte
}

// ListIndex is a simple in-memory index that stores data in a slice.
// It is intended for testing and small datasets.
type ListIndex struct {
	Data []Data
}

// NewListIndex creates a new empty ListIndex.
func NewListIndex() *ListIndex {
	return &ListIndex{
		Data: make([]Data, 0),
	}
}

func (l *ListIndex) Insert(key int64, value []byte) error {
	for i := range l.Data {
		if l.Data[i].Key == key {
			l.Data[i].Val = value
			return nil
		}
	}
	l.Data = append(l.Data, Data{Key: key, Val: value})
	return nil
}

func (l *ListIndex) Get(key int64) ([]byte, error) {
	for _, d := range l.Data {
		if d.Key == key {
			return d.Val, nil
		}
	}
	return nil, errors.New("key not found")
}

func (l *ListIndex) Delete(key int64) error {
	for i, d := range l.Data {
		if d.Key == key {
			l.Data = slices.Delete(l.Data, i, i+1)
			return nil
		}
	}
	return errors.New("key not found")
}

func (l *ListIndex) Range(start, end int64) (index.Iterator, error) {
	return &ListIterator{
		data:  l.Data,
		cur:   -1,
		start: start,
		end:   end,
	}, nil
}

func (l *ListIndex) Close() error { return nil }

type ListIterator struct {
	data  []Data
	cur   int
	start int64
	end   int64
}

// Next advances the iterator to the next key-value pair.
func (it *ListIterator) Next() bool {
	it.cur++
	for it.cur < len(it.data) {
		if it.data[it.cur].Key >= it.start && it.data[it.cur].Key <= it.end {
			return true
		}
		it.cur++
	}
	return false
}

// Key returns the key of the current key-value pair.
func (it *ListIterator) Key() int64 { return it.data[it.cur].Key }

// Value returns the value of the current key-value pair.
func (it *ListIterator) Value() []byte { return it.data[it.cur].Val }

// Error returns the first error encountered by the iterator, if any.
func (it *ListIterator) Error() error { return nil }

// Close releases resources associated with the iterator.
func (it *ListIterator) Close() error { return nil }
