package bplus

import (
	"errors"
	"slices"

	"github.com/btree-query-bench/bmark/index"
	"github.com/btree-query-bench/bmark/persist"
)

var _ index.Index = (*BPlusTree)(nil)

type BPlusNode struct {
	IsLeaf   bool
	Keys     []int64
	Values   [][]byte     // Only populated if IsLeaf == true
	Children []*BPlusNode // Only populated if IsLeaf == false
	Next     *BPlusNode   // Pointer to next leaf for Range scans
}

type BPlusTree struct {
	T    int // Minimum degree (t). Max keys = 2t-1
	Root *BPlusNode
}

func NewBPlusTree(t int) *BPlusTree {
	if t < 2 {
		t = 2
	}
	return &BPlusTree{
		T:    t,
		Root: &BPlusNode{IsLeaf: true},
	}
}

// --- GET (Point Query) ---

func (bt *BPlusTree) Get(key int64) ([]byte, error) {
	node := bt.findLeaf(bt.Root, key)
	idx, found := slices.BinarySearch(node.Keys, key)
	if !found {
		return nil, errors.New("key not found")
	}
	return node.Values[idx], nil
}

func (bt *BPlusTree) findLeaf(curr *BPlusNode, key int64) *BPlusNode {
	for !curr.IsLeaf {
		i := 0
		for i < len(curr.Keys) && key >= curr.Keys[i] {
			i++
		}
		curr = curr.Children[i]
	}
	return curr
}

// --- INSERT ---

func (bt *BPlusTree) Insert(key int64, value []byte) error {
	root := bt.Root
	// If root is full, tree grows in height
	if len(root.Keys) == (2*bt.T - 1) {
		newRoot := &BPlusNode{IsLeaf: false, Children: []*BPlusNode{root}}
		bt.splitChild(newRoot, 0)
		bt.Root = newRoot
	}
	bt.insertNonFull(bt.Root, key, value)
	return nil
}

func (bt *BPlusTree) insertNonFull(x *BPlusNode, k int64, v []byte) {
	if x.IsLeaf {
		idx, found := slices.BinarySearch(x.Keys, k)
		if found {
			x.Values[idx] = v // Update existing
			return
		}
		x.Keys = slices.Insert(x.Keys, idx, k)
		x.Values = slices.Insert(x.Values, idx, v)
	} else {
		i := 0
		for i < len(x.Keys) && k >= x.Keys[i] {
			i++
		}
		if len(x.Children[i].Keys) == (2*bt.T - 1) {
			bt.splitChild(x, i)
			if k >= x.Keys[i] {
				i++
			}
		}
		bt.insertNonFull(x.Children[i], k, v)
	}
}

func (bt *BPlusTree) splitChild(x *BPlusNode, i int) {
	t := bt.T
	y := x.Children[i]
	z := &BPlusNode{IsLeaf: y.IsLeaf}

	if y.IsLeaf {
		// B+ Leaf Split: The first key of the new leaf is copied to parent
		z.Keys = append([]int64{}, y.Keys[t-1:]...)
		z.Values = append([][]byte{}, y.Values[t-1:]...)
		z.Next = y.Next
		y.Next = z

		y.Keys = y.Keys[:t-1]
		y.Values = y.Values[:t-1]

		x.Keys = slices.Insert(x.Keys, i, z.Keys[0])
	} else {
		// B+ Internal Split: Middle key is pushed to parent and removed from child
		z.Keys = append([]int64{}, y.Keys[t:]...)
		z.Children = append([]*BPlusNode{}, y.Children[t:]...)

		midKey := y.Keys[t-1]
		y.Keys = y.Keys[:t-1]
		y.Children = y.Children[:t]

		x.Keys = slices.Insert(x.Keys, i, midKey)
	}
	x.Children = slices.Insert(x.Children, i+1, z)
}

// --- DELETE (Simplified Rebalancing) ---

func (bt *BPlusTree) Delete(key int64) error {
	node := bt.findLeaf(bt.Root, key)
	idx, found := slices.BinarySearch(node.Keys, key)
	if !found {
		return errors.New("key not found")
	}
	node.Keys = slices.Delete(node.Keys, idx, idx+1)
	node.Values = slices.Delete(node.Values, idx, idx+1)
	// Note: In a production B+ Tree, you would check if len(Keys) < t-1
	// and perform borrowing or merging with siblings.
	return nil
}

// --- RANGE (The Iterator) ---

func (bt *BPlusTree) Range(start, end int64) (index.Iterator, error) {
	return &BPlusIterator{
		curr:  bt.findLeaf(bt.Root, start),
		i:     0,
		start: start,
		end:   end,
	}, nil
}

type BPlusIterator struct {
	curr       *BPlusNode
	i          int
	start, end int64
	key        int64
	val        []byte
}

func (it *BPlusIterator) Next() bool {
	for it.curr != nil {
		for it.i < len(it.curr.Keys) {
			k := it.curr.Keys[it.i]
			if k > it.end {
				return false
			}
			if k >= it.start {
				it.key = k
				it.val = it.curr.Values[it.i]
				it.i++
				return true
			}
			it.i++
		}
		// Follow the leaf chain
		it.curr = it.curr.Next
		it.i = 0
	}
	return false
}

func (it *BPlusIterator) Key() int64    { return it.key }
func (it *BPlusIterator) Value() []byte { return it.val }
func (it *BPlusIterator) Error() error  { return nil }
func (it *BPlusIterator) Close() error  { return nil }

func (bt *BPlusTree) SaveTo(path string) error   { return persist.Save(path, bt) }
func (bt *BPlusTree) LoadFrom(path string) error { return persist.Load(path, bt) }
func (bt *BPlusTree) Close() error               { return nil }
