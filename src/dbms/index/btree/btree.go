// Package btree implements a plain B-tree backed by the shared tree engine.
//
// Cell format (same for all nodes):
//
//	[0-3]   uint32  left child page ID
//	[4-11]  int64   key
//	[12-13] uint16  value length
//	[14+]   []byte  value
//
// Internal nodes store the separator key and its value (which is promoted
// during splits). Leaves store the actual user key/value pairs.
// Range iteration uses an explicit stack to perform an in-order traversal.
package btree

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/btpage"
	"github.com/btree-query-bench/bmark/dbms/index/shared"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const cellHeader = 4 + 8 + 2 // leftChild + key + valLen

type BTreeAcc struct{}

func (BTreeAcc) CellSize(_ bool, value []byte) int { return cellHeader + len(value) }

func (BTreeAcc) ReadCell(p *pager.Page, i int, _ bool) (int64, []byte, uint32) {
	off := int(btpage.CellPtr(p, i))
	lc := binary.LittleEndian.Uint32(p[off : off+4])
	key := int64(binary.LittleEndian.Uint64(p[off+4 : off+12]))
	vl := int(binary.LittleEndian.Uint16(p[off+12 : off+14]))
	val := make([]byte, vl)
	copy(val, p[off+14:off+14+vl])
	return key, val, lc
}

func (BTreeAcc) WriteCell(p *pager.Page, off int, key int64, value []byte, leftChild uint32, _ bool) {
	binary.LittleEndian.PutUint32(p[off:off+4], leftChild)
	binary.LittleEndian.PutUint64(p[off+4:off+12], uint64(key))
	binary.LittleEndian.PutUint16(p[off+12:off+14], uint16(len(value)))
	copy(p[off+14:], value)
}

func (BTreeAcc) OverwriteValue(p *pager.Page, i int, newVal []byte, _ bool) {
	off := int(btpage.CellPtr(p, i)) + 12
	binary.LittleEndian.PutUint16(p[off:off+2], uint16(len(newVal)))
	copy(p[off+2:], newVal)
}

func (BTreeAcc) CopyUpLeaves() bool { return false }

func (BTreeAcc) LinkLeaves(_, _ *pager.Page, _, _ uint32) {}

type BTree struct{ shared.Tree }

func Open(path string, cachePages int) (*BTree, error) {
	pg, err := pager.Open(path+".bt", cachePages)
	if err != nil {
		return nil, err
	}
	t := &BTree{shared.Tree{Pg: pg, Acc: BTreeAcc{}}}
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

func (t *BTree) Get(key int64) ([]byte, error)      { return t.Tree.Get(key) }
func (t *BTree) Insert(key int64, val []byte) error { return t.Tree.Insert(key, val) }
func (t *BTree) Delete(_ int64) error               { return nil } // TODO
func (t *BTree) Close() error {
	_ = t.WriteHeader()
	return t.Pg.Close()
}

type frame struct {
	id          uint64
	idx         int
	subtreeDone bool // true after the left subtree of idx has been fully visited
}

type RangeIterator struct {
	tree  *BTree
	end   int64
	stack []frame
	k     int64
	v     []byte
	err   error
}

func (t *BTree) Range(start, end int64) (index.Iterator, error) {
	it := &RangeIterator{tree: t, end: end}
	curr := uint64(t.RootID)
	for {
		p, err := t.Pg.Read(curr)
		if err != nil {
			return nil, err
		}
		n := btpage.NumCells(p)
		leaf := p[btpage.OffType] == btpage.TypeLeaf
		idx := shared.FindIdx(p, start, n, t.Acc, leaf)
		it.stack = append(it.stack, frame{curr, idx, false})
		if leaf {
			break
		}
		curr = uint64(shared.ChildAt(p, idx, n, t.Acc))
	}
	return it, nil
}

func (it *RangeIterator) Next() bool {
	for len(it.stack) > 0 {
		top := len(it.stack) - 1
		f := it.stack[top]
		p, err := it.tree.Pg.Read(f.id)
		if err != nil {
			it.err = err
			return false
		}
		n := btpage.NumCells(p)
		leaf := p[btpage.OffType] == btpage.TypeLeaf

		if leaf {
			if f.idx < n {
				k, v, _ := it.tree.Acc.ReadCell(p, f.idx, true)
				if k > it.end {
					return false
				}
				it.k, it.v = k, v
				it.stack[top].idx++
				return true
			}
			it.stack = it.stack[:top]
			if top > 0 {
				it.stack[top-1].subtreeDone = true
			}
			continue
		}

		if !f.subtreeDone {
			childID := uint64(shared.ChildAt(p, f.idx, n, it.tree.Acc))
			it.stack = append(it.stack, frame{childID, 0, false})
			continue
		}

		if f.idx < n {
			k, v, _ := it.tree.Acc.ReadCell(p, f.idx, false)
			if k > it.end {
				return false
			}
			it.k, it.v = k, v
			it.stack[top].idx++
			it.stack[top].subtreeDone = false
			return true
		}

		it.stack = it.stack[:top]
		if top > 0 {
			it.stack[top-1].subtreeDone = true
		}
	}
	return false
}

func (it *RangeIterator) Key() int64    { return it.k }
func (it *RangeIterator) Value() []byte { return it.v }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
