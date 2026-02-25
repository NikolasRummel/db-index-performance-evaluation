// Package bptree implements a B+-tree backed by the shared tree engine.
//
// Internal cell format:
//
//	[0-3]   uint32  left child page ID
//	[4-11]  int64   key
//
// Leaf cell format:
//
//	[0-7]   int64   key
//	[8-9]   uint16  value length
//	[10+]   []byte  value
//
// Internal nodes carry no values — only keys and child pointers.
// Leaf nodes are linked via nextLeaf for O(1) range-scan advancement.
// Leaf splits use copy-up: the median key is copied into the parent but
// remains in the right leaf as well.
package bptree

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/btpage"
	"github.com/btree-query-bench/bmark/dbms/index/shared"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const (
	internalCellSize = 4 + 8 // leftChild + key
	leafCellHeader   = 8 + 2 // key + valLen
)

type BPTreeAcc struct{}

func (BPTreeAcc) CellSize(isLeaf bool, value []byte) int {
	if isLeaf {
		return leafCellHeader + len(value)
	}
	return internalCellSize
}

func (BPTreeAcc) ReadCell(p *pager.Page, i int, isLeaf bool) (int64, []byte, uint32) {
	off := int(btpage.CellPtr(p, i))
	if isLeaf {
		key := int64(binary.LittleEndian.Uint64(p[off : off+8]))
		vl := int(binary.LittleEndian.Uint16(p[off+8 : off+10]))
		val := make([]byte, vl)
		copy(val, p[off+10:off+10+vl])
		return key, val, 0 // no left-child in leaf cells
	}
	lc := binary.LittleEndian.Uint32(p[off : off+4])
	key := int64(binary.LittleEndian.Uint64(p[off+4 : off+12]))
	return key, nil, lc // no value in internal cells
}

func (BPTreeAcc) WriteCell(p *pager.Page, off int, key int64, value []byte, leftChild uint32, isLeaf bool) {
	if isLeaf {
		binary.LittleEndian.PutUint64(p[off:off+8], uint64(key))
		binary.LittleEndian.PutUint16(p[off+8:off+10], uint16(len(value)))
		copy(p[off+10:], value)
		return
	}
	binary.LittleEndian.PutUint32(p[off:off+4], leftChild)
	binary.LittleEndian.PutUint64(p[off+4:off+12], uint64(key))
}

func (BPTreeAcc) OverwriteValue(p *pager.Page, i int, newVal []byte, isLeaf bool) {
	if !isLeaf {
		return // internal nodes have no value to overwrite
	}
	off := int(btpage.CellPtr(p, i)) + 8
	binary.LittleEndian.PutUint16(p[off:off+2], uint16(len(newVal)))
	copy(p[off+2:], newVal)
}

func (BPTreeAcc) CopyUpLeaves() bool { return true }

func (BPTreeAcc) LinkLeaves(left, right *pager.Page, newRightID uint32, oldNext uint32) {
	btpage.SetNextLeaf(left, newRightID)
	btpage.SetNextLeaf(right, oldNext)
}

// ─── BPTree ───────────────────────────────────────────────────────────────────

type BPTree struct{ shared.Tree }

func Open(path string, cachePages int) (*BPTree, error) {
	pg, err := pager.Open(path+".bpt", cachePages)
	if err != nil {
		return nil, err
	}
	t := &BPTree{shared.Tree{Pg: pg, Acc: BPTreeAcc{}}}
	if pg.PageCount() <= 2 {
		_, _ = pg.Allocate() // page 1: file header
		rootID, _ := pg.Allocate()
		t.RootID = uint32(rootID)
		p := new(pager.Page)
		btpage.InitPage(p, btpage.TypeLeaf)
		_ = pg.Write(rootID, p)
		_ = t.WriteHeader()
	} else {
		_ = t.ReadHeader()
	}
	return t, nil
}

func (t *BPTree) Get(key int64) ([]byte, error) {
	// B+-tree Get: descend to leaf, then check there.
	leafID, err := t.FindLeaf(key)
	if err != nil {
		return nil, err
	}
	p, err := t.Pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	n := btpage.NumCells(p)
	idx := shared.FindIdx(p, key, n, t.Acc, true)
	if idx < n {
		k, val, _ := t.Acc.ReadCell(p, idx, true)
		if k == key {
			return val, nil
		}
	}
	return nil, nil
}

func (t *BPTree) Insert(key int64, val []byte) error { return t.Tree.Insert(key, val) }
func (t *BPTree) Delete(_ int64) error               { return nil } // TODO
func (t *BPTree) Close() error {
	_ = t.WriteHeader()
	return t.Pg.Close()
}

type RangeIterator struct {
	tree   *BPTree
	end    int64
	leafID uint64
	idx    int
	k      int64
	v      []byte
	err    error
}

func (t *BPTree) Range(start, end int64) (index.Iterator, error) {
	leafID, err := t.FindLeaf(start)
	if err != nil {
		return nil, err
	}
	p, err := t.Pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	idx := shared.FindIdx(p, start, btpage.NumCells(p), t.Acc, true)
	return &RangeIterator{tree: t, end: end, leafID: leafID, idx: idx}, nil
}

func (it *RangeIterator) Next() bool {
	for it.leafID != uint64(btpage.InvalidPage) {
		p, err := it.tree.Pg.Read(it.leafID)
		if err != nil {
			it.err = err
			return false
		}
		n := btpage.NumCells(p)
		if it.idx < n {
			k, v, _ := it.tree.Acc.ReadCell(p, it.idx, true)
			if k > it.end {
				return false
			}
			it.k, it.v = k, v
			it.idx++
			return true
		}
		it.leafID = uint64(btpage.NextLeaf(p))
		it.idx = 0
	}
	return false
}

func (it *RangeIterator) Key() int64    { return it.k }
func (it *RangeIterator) Value() []byte { return it.v }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
