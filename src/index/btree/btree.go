package btree

import (
	"errors"
	"slices"

	"github.com/btree-query-bench/bmark/index"
)

type BTreeNode struct {
	Leaf     bool
	Keys     []int64
	Values   [][]byte
	Children []*BTreeNode
}

type BTree struct {
	T    int
	Root *BTreeNode
}

func NewBTree(t int) *BTree {
	if t < 2 {
		t = 2
	}
	return &BTree{T: t, Root: &BTreeNode{Leaf: true}}
}

func (bt *BTree) Get(key int64) ([]byte, error) {
	return bt.search(bt.Root, key)
}

func (bt *BTree) search(x *BTreeNode, key int64) ([]byte, error) {
	i, found := slices.BinarySearch(x.Keys, key)
	if found {
		return x.Values[i], nil
	}
	if x.Leaf {
		return nil, errors.New("key not found")
	}
	return bt.search(x.Children[i], key)
}

func (bt *BTree) Insert(key int64, value []byte) error {
	root := bt.Root
	if len(root.Keys) == (2*bt.T - 1) {
		newRoot := &BTreeNode{Children: []*BTreeNode{root}}
		bt.splitChild(newRoot, 0)
		bt.Root = newRoot
	}
	bt.insertNonFull(bt.Root, key, value)
	return nil
}

func (bt *BTree) insertNonFull(x *BTreeNode, k int64, v []byte) {
	if x.Leaf {
		idx, found := slices.BinarySearch(x.Keys, k)
		if found {
			x.Values[idx] = v
			return
		}
		x.Keys = slices.Insert(x.Keys, idx, k)
		x.Values = slices.Insert(x.Values, idx, v)
	} else {
		i := 0
		for i < len(x.Keys) && k > x.Keys[i] {
			i++
		}
		if len(x.Children[i].Keys) == (2*bt.T - 1) {
			bt.splitChild(x, i)
			if k > x.Keys[i] {
				i++
			}
		}
		bt.insertNonFull(x.Children[i], k, v)
	}
}

func (bt *BTree) splitChild(x *BTreeNode, i int) {
	t := bt.T
	y := x.Children[i]
	z := &BTreeNode{Leaf: y.Leaf}
	z.Keys = append(z.Keys, y.Keys[t:]...)
	z.Values = append(z.Values, y.Values[t:]...)
	if !y.Leaf {
		z.Children = append(z.Children, y.Children[t:]...)
	}

	midKey, midVal := y.Keys[t-1], y.Values[t-1]
	y.Keys, y.Values = y.Keys[:t-1], y.Values[:t-1]
	if !y.Leaf {
		y.Children = y.Children[:t]
	}

	x.Keys = slices.Insert(x.Keys, i, midKey)
	x.Values = slices.Insert(x.Values, i, midVal)
	x.Children = slices.Insert(x.Children, i+1, z)
}

func (bt *BTree) Delete(key int64) error {
	bt.delete(bt.Root, key)
	if len(bt.Root.Keys) == 0 && !bt.Root.Leaf {
		bt.Root = bt.Root.Children[0]
	}
	return nil
}

func (bt *BTree) delete(x *BTreeNode, k int64) {
	idx, found := slices.BinarySearch(x.Keys, k)
	if found {
		if x.Leaf {
			x.Keys = slices.Delete(x.Keys, idx, idx+1)
			x.Values = slices.Delete(x.Values, idx, idx+1)
		} else {
			bt.deleteInternal(x, idx)
		}
	} else if !x.Leaf {
		child := x.Children[idx]
		if len(child.Keys) < bt.T {
			bt.fill(x, idx)
		}
		if idx > len(x.Keys) {
			bt.delete(x.Children[idx-1], k)
		} else {
			bt.delete(x.Children[idx], k)
		}
	}
}

func (bt *BTree) deleteInternal(x *BTreeNode, i int) {
	k, y, z := x.Keys[i], x.Children[i], x.Children[i+1]
	if len(y.Keys) >= bt.T {
		pk, pv := bt.getPred(y)
		x.Keys[i], x.Values[i] = pk, pv
		bt.delete(y, pk)
	} else if len(z.Keys) >= bt.T {
		sk, sv := bt.getSucc(z)
		x.Keys[i], x.Values[i] = sk, sv
		bt.delete(z, sk)
	} else {
		bt.merge(x, i)
		bt.delete(y, k)
	}
}

func (bt *BTree) getPred(x *BTreeNode) (int64, []byte) {
	for !x.Leaf {
		x = x.Children[len(x.Keys)]
	}
	return x.Keys[len(x.Keys)-1], x.Values[len(x.Values)-1]
}

func (bt *BTree) getSucc(x *BTreeNode) (int64, []byte) {
	for !x.Leaf {
		x = x.Children[0]
	}
	return x.Keys[0], x.Values[0]
}

func (bt *BTree) fill(x *BTreeNode, i int) {
	if i != 0 && len(x.Children[i-1].Keys) >= bt.T {
		bt.borrowPrev(x, i)
	} else if i != len(x.Keys) && len(x.Children[i+1].Keys) >= bt.T {
		bt.borrowNext(x, i)
	} else {
		if i != len(x.Keys) {
			bt.merge(x, i)
		} else {
			bt.merge(x, i-1)
		}
	}
}

func (bt *BTree) borrowPrev(x *BTreeNode, i int) {
	c, s := x.Children[i], x.Children[i-1]
	c.Keys = slices.Insert(c.Keys, 0, x.Keys[i-1])
	c.Values = slices.Insert(c.Values, 0, x.Values[i-1])
	if !c.Leaf {
		c.Children = slices.Insert(c.Children, 0, s.Children[len(s.Keys)])
		s.Children = s.Children[:len(s.Keys)]
	}
	x.Keys[i-1], x.Values[i-1] = s.Keys[len(s.Keys)-1], s.Values[len(s.Keys)-1]
	s.Keys, s.Values = s.Keys[:len(s.Keys)-1], s.Values[:len(s.Values)-1]
}

func (bt *BTree) borrowNext(x *BTreeNode, i int) {
	c, s := x.Children[i], x.Children[i+1]
	c.Keys, c.Values = append(c.Keys, x.Keys[i]), append(c.Values, x.Values[i])
	if !c.Leaf {
		c.Children = append(c.Children, s.Children[0])
		s.Children = slices.Delete(s.Children, 0, 1)
	}
	x.Keys[i], x.Values[i] = s.Keys[0], s.Values[0]
	s.Keys, s.Values = s.Keys[1:], s.Values[1:]
}

func (bt *BTree) merge(x *BTreeNode, i int) {
	y, z := x.Children[i], x.Children[i+1]
	y.Keys, y.Values = append(y.Keys, x.Keys[i]), append(y.Values, x.Values[i])
	y.Keys, y.Values = append(y.Keys, z.Keys...), append(y.Values, z.Values...)
	if !y.Leaf {
		y.Children = append(y.Children, z.Children...)
	}
	x.Keys, x.Values = slices.Delete(x.Keys, i, i+1), slices.Delete(x.Values, i, i+1)
	x.Children = slices.Delete(x.Children, i+1, i+2)
}

func (bt *BTree) Range(start, end int64) (index.Iterator, error) {
	it := &BTreeIterator{idx: -1}
	bt.collect(bt.Root, start, end, it)
	return it, nil
}

func (bt *BTree) collect(x *BTreeNode, s, e int64, it *BTreeIterator) {
	for i := 0; i < len(x.Keys); i++ {
		if !x.Leaf {
			bt.collect(x.Children[i], s, e, it)
		}
		if x.Keys[i] >= s && x.Keys[i] <= e {
			it.data = append(it.data, struct {
				k int64
				v []byte
			}{x.Keys[i], x.Values[i]})
		}
	}
	if !x.Leaf {
		bt.collect(x.Children[len(x.Keys)], s, e, it)
	}
}

type BTreeIterator struct {
	data []struct {
		k int64
		v []byte
	}
	idx int
}

func (it *BTreeIterator) Next() bool      { it.idx++; return it.idx < len(it.data) }
func (it *BTreeIterator) Key() int64      { return it.data[it.idx].k }
func (it *BTreeIterator) Value() []byte   { return it.data[it.idx].v }
func (it *BTreeIterator) Error() error    { return nil }
func (it *BTreeIterator) Close() error    { return nil }
func (bt *BTree) SaveTo(p string) error   { return nil }
func (bt *BTree) LoadFrom(p string) error { return nil }
func (bt *BTree) Close() error            { return nil }
