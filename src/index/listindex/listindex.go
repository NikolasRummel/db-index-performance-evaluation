package listindex

import (
	"errors"
	"slices"

	"github.com/btree-query-bench/bmark/index"
	"github.com/btree-query-bench/bmark/persist"
)

var _ index.Index = (*ListIndex)(nil)

type Data struct {
	Key int64
	Val []byte
}

type ListIndex struct {
	Data []Data
}

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

func (l *ListIndex) SaveTo(path string) error   { return persist.Save(path, l.Data) }
func (l *ListIndex) LoadFrom(path string) error { return persist.Load(path, &l.Data) }
func (l *ListIndex) Close() error               { return nil }

type ListIterator struct {
	data  []Data
	cur   int
	start int64
	end   int64
}

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

func (it *ListIterator) Key() int64    { return it.data[it.cur].Key }
func (it *ListIterator) Value() []byte { return it.data[it.cur].Val }
func (it *ListIterator) Error() error  { return nil }
func (it *ListIterator) Close() error  { return nil }
