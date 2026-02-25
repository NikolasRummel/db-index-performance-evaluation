// Package bptree implements a B+ tree.
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
// Internal nodes store no values — only keys and child pointers.
// Leaf nodes are linked via nextLeaf for fast range scans.
// The rightmost child is stored in the page header for internal nodes.
package bptree

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/btpage"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const internalCellSize = 4 + 8 // leftChild + key (fixed, no value)

// ─── Internal cell read/write ─────────────────────────────────────────────────

func readInternalCell(p *pager.Page, i int) (key int64, leftChild uint32) {
	off := int(btpage.CellPtr(p, i))
	leftChild = binary.LittleEndian.Uint32(p[off : off+4])
	key = int64(binary.LittleEndian.Uint64(p[off+4 : off+12]))
	return
}

func writeInternalCell(p *pager.Page, off int, key int64, leftChild uint32) {
	binary.LittleEndian.PutUint32(p[off:off+4], leftChild)
	binary.LittleEndian.PutUint64(p[off+4:off+12], uint64(key))
}

func appendInternalCell(p *pager.Page, key int64, leftChild uint32) {
	n := btpage.NumCells(p)
	off := btpage.AllocCell(p, internalCellSize)
	writeInternalCell(p, off, key, leftChild)
	btpage.SetCellPtr(p, n, uint16(off))
	btpage.SetNumCells(p, n+1)
}

func internalChildAt(p *pager.Page, idx, n int) uint32 {
	if idx == n {
		return btpage.Rightmost(p)
	}
	_, lc := readInternalCell(p, idx)
	return lc
}

func findInternalIdx(p *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _ := readInternalCell(p, m)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// ─── Leaf cell read/write ─────────────────────────────────────────────────────

func leafCellSize(value []byte) int { return 8 + 2 + len(value) }

func readLeafCell(p *pager.Page, i int) (key int64, value []byte) {
	off := int(btpage.CellPtr(p, i))
	key = int64(binary.LittleEndian.Uint64(p[off : off+8]))
	vl := int(binary.LittleEndian.Uint16(p[off+8 : off+10]))
	value = make([]byte, vl)
	copy(value, p[off+10:off+10+vl])
	return
}

func writeLeafCell(p *pager.Page, off int, key int64, value []byte) {
	binary.LittleEndian.PutUint64(p[off:off+8], uint64(key))
	binary.LittleEndian.PutUint16(p[off+8:off+10], uint16(len(value)))
	copy(p[off+10:], value)
}

func deleteLeafCell(p *pager.Page, i int) {
	n := btpage.NumCells(p)
	for j := i; j < n-1; j++ {
		btpage.SetCellPtr(p, j, btpage.CellPtr(p, j+1))
	}
	btpage.SetNumCells(p, n-1)
}

func findLeafIdx(p *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _ := readLeafCell(p, m)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// ─── BPTree ───────────────────────────────────────────────────────────────────

type BPTree struct {
	pg     *pager.Pager
	rootID uint32
}

func Open(path string, cachePages int) (*BPTree, error) {
	pg, err := pager.Open(path+".bpt", cachePages)
	if err != nil {
		return nil, err
	}
	t := &BPTree{pg: pg}
	if pg.PageCount() <= 2 {
		_, _ = pg.Allocate() // page 1: file header
		rootID, _ := pg.Allocate()
		t.rootID = uint32(rootID)
		p := new(pager.Page)
		btpage.InitPage(p, btpage.TypeLeaf)
		_ = pg.Write(rootID, p)
		_ = t.writeHeader()
	} else {
		_ = t.readHeader()
	}
	return t, nil
}

// ─── Get ──────────────────────────────────────────────────────────────────────

func (t *BPTree) Get(key int64) ([]byte, error) {
	leafID, err := t.findLeaf(key)
	if err != nil {
		return nil, err
	}
	p, err := t.pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	n := btpage.NumCells(p)
	idx := findLeafIdx(p, key, n)
	if idx < n {
		k, val := readLeafCell(p, idx)
		if k == key {
			return val, nil
		}
	}
	return nil, nil
}

// ─── Insert ───────────────────────────────────────────────────────────────────

func (t *BPTree) Insert(key int64, value []byte) error {
	mk, rightID, split, err := t.insertRec(uint64(t.rootID), key, value)
	if err != nil {
		return err
	}
	if !split {
		return nil
	}
	newRoot, _ := t.pg.Allocate()
	p := new(pager.Page)
	btpage.InitPage(p, btpage.TypeInternal)
	btpage.SetRightmost(p, uint32(rightID))
	appendInternalCell(p, mk, t.rootID)
	_ = t.pg.Write(newRoot, p)
	t.rootID = uint32(newRoot)
	return t.writeHeader()
}

// insertRec returns (promotedKey, rightPageID, didSplit, error).
func (t *BPTree) insertRec(id uint64, key int64, value []byte) (int64, uint64, bool, error) {
	p, err := t.pg.Read(id)
	if err != nil {
		return 0, 0, false, err
	}
	if p[btpage.OffType] == btpage.TypeLeaf {
		return t.insertLeaf(id, p, key, value)
	}
	n := btpage.NumCells(p)
	idx := findInternalIdx(p, key, n)
	childID := uint64(internalChildAt(p, idx, n))
	mk, rc, split, err := t.insertRec(childID, key, value)
	if err != nil || !split {
		return 0, 0, false, err
	}
	p, err = t.pg.Read(id)
	if err != nil {
		return 0, 0, false, err
	}
	n = btpage.NumCells(p)
	idx = findInternalIdx(p, mk, n)
	return t.insertInternal(id, p, n, idx, mk, rc)
}

func (t *BPTree) insertLeaf(id uint64, p *pager.Page, key int64, value []byte) (int64, uint64, bool, error) {
	n := btpage.NumCells(p)
	idx := findLeafIdx(p, key, n)

	if idx < n {
		if k, oldVal := readLeafCell(p, idx); k == key {
			if len(value) <= len(oldVal) {
				writeLeafCell(p, int(btpage.CellPtr(p, idx)), key, value)
				return 0, 0, false, t.pg.Write(id, p)
			}
			deleteLeafCell(p, idx)
			n--
		}
	}

	if btpage.FreeSpace(p, n) >= leafCellSize(value) {
		for i := n; i > idx; i-- {
			btpage.SetCellPtr(p, i, btpage.CellPtr(p, i-1))
		}
		off := btpage.AllocCell(p, leafCellSize(value))
		writeLeafCell(p, off, key, value)
		btpage.SetCellPtr(p, idx, uint16(off))
		btpage.SetNumCells(p, n+1)
		return 0, 0, false, t.pg.Write(id, p)
	}

	return t.splitLeaf(id, p, n, idx, key, value)
}

func (t *BPTree) splitLeaf(id uint64, p *pager.Page, n, idx int, key int64, value []byte) (int64, uint64, bool, error) {
	type lc struct {
		key   int64
		value []byte
	}
	all := make([]lc, n+1)
	for i := 0; i < n; i++ {
		k, v := readLeafCell(p, i)
		all[i] = lc{k, v}
	}
	copy(all[idx+1:], all[idx:n])
	all[idx] = lc{key, value}

	mid := (n + 1) / 2
	newID, _ := t.pg.Allocate()
	right := new(pager.Page)
	btpage.InitPage(right, btpage.TypeLeaf)

	// Link: left -> right -> old next
	oldNext := btpage.NextLeaf(p)
	btpage.SetNextLeaf(right, oldNext)
	btpage.SetNextLeaf(p, uint32(newID))

	btpage.InitPage(p, btpage.TypeLeaf)
	btpage.SetNextLeaf(p, uint32(newID))
	for i := 0; i < mid; i++ {
		off := btpage.AllocCell(p, leafCellSize(all[i].value))
		writeLeafCell(p, off, all[i].key, all[i].value)
		btpage.SetCellPtr(p, i, uint16(off))
	}
	btpage.SetNumCells(p, mid)

	for i := mid; i <= n; i++ {
		off := btpage.AllocCell(right, leafCellSize(all[i].value))
		writeLeafCell(right, off, all[i].key, all[i].value)
		btpage.SetCellPtr(right, i-mid, uint16(off))
	}
	btpage.SetNumCells(right, n+1-mid)

	_ = t.pg.Write(id, p)
	_ = t.pg.Write(newID, right)
	return all[mid].key, newID, true, nil // copy-up
}

func (t *BPTree) insertInternal(id uint64, p *pager.Page, n, idx int, key int64, rightChild uint64) (int64, uint64, bool, error) {
	if btpage.FreeSpace(p, n) >= internalCellSize {
		for i := n; i > idx; i-- {
			btpage.SetCellPtr(p, i, btpage.CellPtr(p, i-1))
		}
		off := btpage.AllocCell(p, internalCellSize)
		writeInternalCell(p, off, key, internalChildAt(p, idx, n))
		btpage.SetCellPtr(p, idx, uint16(off))
		if idx == n {
			btpage.SetRightmost(p, uint32(rightChild))
		} else {
			off1 := int(btpage.CellPtr(p, idx+1))
			binary.LittleEndian.PutUint32(p[off1:off1+4], uint32(rightChild))
		}
		btpage.SetNumCells(p, n+1)
		return 0, 0, false, t.pg.Write(id, p)
	}

	return t.splitInternal(id, p, n, idx, key, rightChild)
}

func (t *BPTree) splitInternal(id uint64, p *pager.Page, n, idx int, key int64, rightChild uint64) (int64, uint64, bool, error) {
	type ic struct {
		key       int64
		leftChild uint32
	}
	all := make([]ic, n+1)
	for i := 0; i < n; i++ {
		k, lc := readInternalCell(p, i)
		all[i] = ic{k, lc}
	}
	copy(all[idx+1:], all[idx:n])
	all[idx] = ic{key, internalChildAt(p, idx, n)}
	if idx+1 <= n {
		all[idx+1].leftChild = uint32(rightChild)
	}
	oldRightmost := btpage.Rightmost(p)
	if idx == n {
		all[n].leftChild = oldRightmost
		oldRightmost = uint32(rightChild)
	}

	mid := (n + 1) / 2
	median := all[mid]

	newID, _ := t.pg.Allocate()
	right := new(pager.Page)

	btpage.InitPage(p, btpage.TypeInternal)
	btpage.SetRightmost(p, median.leftChild)
	for i := 0; i < mid; i++ {
		appendInternalCell(p, all[i].key, all[i].leftChild)
	}

	btpage.InitPage(right, btpage.TypeInternal)
	btpage.SetRightmost(right, oldRightmost)
	for i := mid + 1; i <= n; i++ {
		appendInternalCell(right, all[i].key, all[i].leftChild)
	}

	_ = t.pg.Write(id, p)
	_ = t.pg.Write(newID, right)
	return median.key, newID, true, nil // push-up (median leaves internal)
}

// ─── Find leaf ────────────────────────────────────────────────────────────────

func (t *BPTree) findLeaf(key int64) (uint64, error) {
	curr := uint64(t.rootID)
	for {
		p, err := t.pg.Read(curr)
		if err != nil {
			return 0, err
		}
		if p[btpage.OffType] == btpage.TypeLeaf {
			return curr, nil
		}
		n := btpage.NumCells(p)
		idx := findInternalIdx(p, key, n)
		curr = uint64(internalChildAt(p, idx, n))
	}
}

// ─── Header ───────────────────────────────────────────────────────────────────

func (t *BPTree) writeHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(p[:4], t.rootID)
	return t.pg.Write(1, p)
}

func (t *BPTree) readHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	t.rootID = binary.LittleEndian.Uint32(p[:4])
	return nil
}

func (t *BPTree) Delete(k int64) error { return nil }

func (t *BPTree) Close() error {
	_ = t.writeHeader()
	return t.pg.Close()
}

// ─── Range Iterator ───────────────────────────────────────────────────────────

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
	leafID, err := t.findLeaf(start)
	if err != nil {
		return nil, err
	}
	p, err := t.pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	idx := findLeafIdx(p, start, btpage.NumCells(p))
	return &RangeIterator{tree: t, end: end, leafID: leafID, idx: idx}, nil
}

func (it *RangeIterator) Next() bool {
	for it.leafID != uint64(btpage.InvalidPage) {
		p, err := it.tree.pg.Read(it.leafID)
		if err != nil {
			it.err = err
			return false
		}
		n := btpage.NumCells(p)
		if it.idx < n {
			k, v := readLeafCell(p, it.idx)
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
