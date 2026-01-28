package lsmtree

import (
	"container/heap"
	"errors"
	"slices"
	"sort"

	"github.com/btree-query-bench/bmark/index"
)

// Ensure LSMTree implements the index.Index interface
var _ index.Index = (*LSMTree)(nil)

type Entry struct {
	Key int64
	Val []byte // nil = Tombstone
}

type Segment struct {
	Data   []Entry
	Filter *BloomFilter
}

type LSMTree struct {
	MemTable  []Entry
	Levels    [][]Segment // Level 0 contains multiple segments, Levels 1+ are merged
	Threshold int         // Max size of MemTable before flush
}

func NewLSM(threshold int) *LSMTree {
	return &LSMTree{
		Threshold: threshold,
		MemTable:  make([]Entry, 0, threshold),
		Levels:    make([][]Segment, 5), // L0 to L4
	}
}

// --- WRITE OPERATIONS ---

func (l *LSMTree) Insert(k int64, v []byte) error {
	l.MemTable = append(l.MemTable, Entry{k, v})
	if len(l.MemTable) >= l.Threshold {
		l.flush()
	}
	return nil
}

func (l *LSMTree) Delete(k int64) error {
	return l.Insert(k, nil)
}

func (l *LSMTree) flush() {
	// Sort MemTable to turn it into an SSTable
	slices.SortFunc(l.MemTable, func(a, b Entry) int {
		return int(a.Key - b.Key)
	})

	// Build Bloom Filter
	filter := NewBloom(len(l.MemTable)*10, 3)
	for _, e := range l.MemTable {
		filter.Add(e.Key)
	}

	// Push to Level 0
	l.Levels[0] = append([]Segment{{Data: l.MemTable, Filter: filter}}, l.Levels[0]...)
	l.MemTable = make([]Entry, 0, l.Threshold)

	// Trigger 10x Leveling Check
	l.checkCompaction(0)
}

func (l *LSMTree) checkCompaction(level int) {
	// If a level has more than 10 segments, merge them into the next level
	if len(l.Levels[level]) >= 10 && level < len(l.Levels)-1 {
		l.compactLevel(level)
	}
}

func (l *LSMTree) compactLevel(level int) {
	var combined []Entry
	for _, s := range l.Levels[level] {
		combined = append(combined, s.Data...)
	}

	// Stable Sort: newer segments are at the beginning of the slice
	sort.SliceStable(combined, func(i, j int) bool {
		return combined[i].Key < combined[j].Key
	})

	var compacted []Entry
	for i := 0; i < len(combined); i++ {
		if i > 0 && combined[i].Key == combined[i-1].Key {
			continue // Keep newest version (Deduplicate)
		}
		compacted = append(compacted, combined[i])
	}

	filter := NewBloom(len(compacted)*10, 3)
	for _, e := range compacted {
		filter.Add(e.Key)
	}

	l.Levels[level+1] = append([]Segment{{Data: compacted, Filter: filter}}, l.Levels[level+1]...)
	l.Levels[level] = make([]Segment, 0)

	l.checkCompaction(level + 1)
}

// --- READ OPERATIONS ---

func (l *LSMTree) Get(key int64) ([]byte, error) {
	// 1. Search MemTable
	for i := len(l.MemTable) - 1; i >= 0; i-- {
		if l.MemTable[i].Key == key {
			if l.MemTable[i].Val == nil {
				return nil, errors.New("deleted")
			}
			return l.MemTable[i].Val, nil
		}
	}

	// 2. Search Levels
	for _, level := range l.Levels {
		for _, s := range level {
			if !s.Filter.Test(key) {
				continue
			}
			idx, found := slices.BinarySearchFunc(s.Data, key, func(e Entry, t int64) int {
				return int(e.Key - t)
			})
			if found {
				if s.Data[idx].Val == nil {
					return nil, errors.New("deleted")
				}
				return s.Data[idx].Val, nil
			}
		}
	}
	return nil, errors.New("not found")
}

// --- RANGE (Using Priority Queue / Heap) ---

func (l *LSMTree) Range(start, end int64) (index.Iterator, error) {
	h := &MergeHeap{}
	heap.Init(h)

	if len(l.MemTable) > 0 {
		heap.Push(h, &HeapItem{data: l.MemTable, index: 0})
	}

	for _, level := range l.Levels {
		for _, seg := range level {
			if len(seg.Data) > 0 {
				heap.Push(h, &HeapItem{data: seg.Data, index: 0})
			}
		}
	}

	var final []Entry
	var lastKey int64 = -1
	var first = true

	for h.Len() > 0 {
		item := heap.Pop(h).(*HeapItem)
		entry := item.data[item.index]

		if entry.Key >= start && entry.Key <= end {
			if first || entry.Key != lastKey {
				if entry.Val != nil {
					final = append(final, entry)
				}
				lastKey = entry.Key
				first = false
			}
		}

		item.index++
		if item.index < len(item.data) {
			heap.Push(h, item)
		} else if entry.Key > end && entry.Key != lastKey {
			// Early exit if we are past the range (only works if segments are fully sorted)
		}
	}

	return &LSMIterator{data: final, idx: -1}, nil
}

// --- HEAP IMPLEMENTATION ---

type HeapItem struct {
	data  []Entry
	index int
}

type MergeHeap []*HeapItem

func (h MergeHeap) Len() int            { return len(h) }
func (h MergeHeap) Less(i, j int) bool  { return h[i].data[h[i].index].Key < h[j].data[h[j].index].Key }
func (h MergeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *MergeHeap) Push(x interface{}) { *h = append(*h, x.(*HeapItem)) }
func (h *MergeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// --- ITERATOR & UNUSED INTERFACE METHODS ---

type LSMIterator struct {
	data []Entry
	idx  int
}

func (it *LSMIterator) Next() bool    { it.idx++; return it.idx < len(it.data) }
func (it *LSMIterator) Key() int64    { return it.data[it.idx].Key }
func (it *LSMIterator) Value() []byte { return it.data[it.idx].Val }
func (it *LSMIterator) Error() error  { return nil }
func (it *LSMIterator) Close() error  { return nil }

func (l *LSMTree) Close() error            { return nil }
func (l *LSMTree) SaveTo(p string) error   { return nil }
func (l *LSMTree) LoadFrom(p string) error { return nil }
