package btree

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/btpage"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const cellHeader = 4 + 8 + 2 // leftChild + key + valLen

func cellSize(value []byte) int { return cellHeader + len(value) }

// --- Internal Cell Helpers ---

func readCell(p *pager.Page, i int) (key int64, value []byte, leftChild uint32) {
	off := int(btpage.CellPtr(p, i))
	leftChild = binary.LittleEndian.Uint32(p[off : off+4])
	key = int64(binary.LittleEndian.Uint64(p[off+4 : off+12]))
	vl := int(binary.LittleEndian.Uint16(p[off+12 : off+14]))
	value = make([]byte, vl)
	copy(value, p[off+14:off+14+vl])
	return
}

func writeCell(p *pager.Page, off int, key int64, value []byte, leftChild uint32) {
	binary.LittleEndian.PutUint32(p[off:off+4], leftChild)
	binary.LittleEndian.PutUint64(p[off+4:off+12], uint64(key))
	binary.LittleEndian.PutUint16(p[off+12:off+14], uint16(len(value)))
	copy(p[off+14:], value)
}

func appendCell(p *pager.Page, key int64, value []byte, leftChild uint32) {
	n := btpage.NumCells(p)
	off := btpage.AllocCell(p, cellSize(value))
	writeCell(p, off, key, value, leftChild)
	btpage.SetCellPtr(p, n, uint16(off))
	btpage.SetNumCells(p, n+1)
}

func overwriteValue(p *pager.Page, i int, value []byte) {
	off := int(btpage.CellPtr(p, i)) + 12
	binary.LittleEndian.PutUint16(p[off:off+2], uint16(len(value)))
	copy(p[off+2:], value)
}

func deleteCell(p *pager.Page, i int) {
	n := btpage.NumCells(p)
	for j := i; j < n-1; j++ {
		btpage.SetCellPtr(p, j, btpage.CellPtr(p, j+1))
	}
	btpage.SetNumCells(p, n-1)
}

func childAt(p *pager.Page, idx, n int) uint32 {
	if idx == n {
		return btpage.Rightmost(p)
	}
	_, _, lc := readCell(p, idx)
	return lc
}

func findIdx(p *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _, _ := readCell(p, m)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// --- BTree Implementation ---

type BTree struct {
	pg     *pager.Pager
	rootID uint32
}

func Open(path string, cachePages int) (*BTree, error) {
	pg, err := pager.Open(path+".bt", cachePages)
	if err != nil {
		return nil, err
	}
	t := &BTree{pg: pg}
	if pg.PageCount() <= 2 {
		_, _ = pg.Allocate() // page 0: pager header; page 1: btree header
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

func (t *BTree) Get(key int64) ([]byte, error) {
	curr := uint64(t.rootID)
	for {
		p, err := t.pg.Read(curr)
		if err != nil {
			return nil, err
		}
		n := btpage.NumCells(p)
		idx := findIdx(p, key, n)
		if idx < n {
			k, val, _ := readCell(p, idx)
			if k == key {
				return val, nil
			}
		}
		if p[btpage.OffType] == btpage.TypeLeaf {
			return nil, nil
		}
		curr = uint64(childAt(p, idx, n))
	}
}

func (t *BTree) Insert(key int64, value []byte) error {
	mk, mv, rightID, split, err := t.insertRec(uint64(t.rootID), key, value)
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
	appendCell(p, mk, mv, t.rootID)
	_ = t.pg.Write(newRoot, p)
	t.rootID = uint32(newRoot)
	return t.writeHeader()
}

func (t *BTree) insertRec(id uint64, key int64, value []byte) (int64, []byte, uint64, bool, error) {
	p, err := t.pg.Read(id)
	if err != nil {
		return 0, nil, 0, false, err
	}
	n := btpage.NumCells(p)
	idx := findIdx(p, key, n)

	if idx < n {
		if k, oldVal, _ := readCell(p, idx); k == key {
			if len(value) <= len(oldVal) {
				overwriteValue(p, idx, value)
				return 0, nil, 0, false, t.pg.Write(id, p)
			}
			deleteCell(p, idx)
			n--
		}
	}

	if p[btpage.OffType] == btpage.TypeLeaf {
		return t.doInsert(id, p, n, idx, key, value, 0)
	}

	childID := uint64(childAt(p, idx, n))
	mk, mv, rc, split, err := t.insertRec(childID, key, value)
	if err != nil || !split {
		return 0, nil, 0, false, err
	}

	p, _ = t.pg.Read(id)
	n = btpage.NumCells(p)
	idx = findIdx(p, mk, n)
	return t.doInsert(id, p, n, idx, mk, mv, rc)
}

type cellData struct {
	key       int64
	value     []byte
	leftChild uint32
}

func (t *BTree) doInsert(id uint64, p *pager.Page, n, idx int, key int64, value []byte, rightChild uint64) (int64, []byte, uint64, bool, error) {
	if btpage.FreeSpace(p, n) >= cellSize(value) {
		for i := n; i > idx; i-- {
			btpage.SetCellPtr(p, i, btpage.CellPtr(p, i-1))
		}
		off := btpage.AllocCell(p, cellSize(value))
		writeCell(p, off, key, value, childAt(p, idx, n))
		btpage.SetCellPtr(p, idx, uint16(off))
		if idx == n {
			btpage.SetRightmost(p, uint32(rightChild))
		} else {
			off1 := int(btpage.CellPtr(p, idx+1))
			binary.LittleEndian.PutUint32(p[off1:off1+4], uint32(rightChild))
		}
		btpage.SetNumCells(p, n+1)
		return 0, nil, 0, false, t.pg.Write(id, p)
	}

	// Split Logic
	all := make([]cellData, n+1)
	for i := 0; i < n; i++ {
		k, v, lc := readCell(p, i)
		all[i] = cellData{k, v, lc}
	}
	copy(all[idx+1:], all[idx:n])
	all[idx] = cellData{key, value, childAt(p, idx, n)}

	oldRightmost := btpage.Rightmost(p)
	if idx == n {
		oldRightmost = uint32(rightChild)
	} else {
		all[idx+1].leftChild = uint32(rightChild)
	}

	mid := (n + 1) / 2
	median := all[mid]

	newID, _ := t.pg.Allocate()
	right := new(pager.Page)

	btpage.InitPage(p, p[btpage.OffType])
	btpage.SetRightmost(p, median.leftChild)
	for i := 0; i < mid; i++ {
		appendCell(p, all[i].key, all[i].value, all[i].leftChild)
	}

	btpage.InitPage(right, p[btpage.OffType])
	btpage.SetRightmost(right, oldRightmost)
	for i := mid + 1; i <= n; i++ {
		appendCell(right, all[i].key, all[i].value, all[i].leftChild)
	}

	_ = t.pg.Write(id, p)
	_ = t.pg.Write(newID, right)
	return median.key, median.value, newID, true, nil
}

// --- Iterator & Range ---

type frame struct {
	id  uint64
	idx int
}

type RangeIterator struct {
	tree        *BTree
	end         int64
	stack       []frame
	k           int64
	v           []byte
	err         error
	subTreeDone bool
}

func (t *BTree) Range(start, end int64) (index.Iterator, error) {
	it := &RangeIterator{tree: t, end: end}
	curr := uint64(t.rootID)
	for {
		p, err := t.pg.Read(curr)
		if err != nil {
			return nil, err
		}
		n := btpage.NumCells(p)
		idx := findIdx(p, start, n)
		it.stack = append(it.stack, frame{curr, idx})
		if p[btpage.OffType] == btpage.TypeLeaf {
			break
		}
		curr = uint64(childAt(p, idx, n))
	}
	return it, nil
}

func (it *RangeIterator) Next() bool {
	for len(it.stack) > 0 {
		f := &it.stack[len(it.stack)-1]
		p, err := it.tree.pg.Read(f.id)
		if err != nil {
			it.err = err
			return false
		}
		n := btpage.NumCells(p)
		isLeaf := p[btpage.OffType] == btpage.TypeLeaf

		if isLeaf {
			if f.idx < n {
				k, v, _ := readCell(p, f.idx)
				if k > it.end {
					return false
				}
				it.k, it.v, f.idx = k, v, f.idx+1
				return true
			}
			it.stack = it.stack[:len(it.stack)-1]
			it.subTreeDone = true
			continue
		}

		if !it.subTreeDone {
			childID := uint64(childAt(p, f.idx, n))
			it.stack = append(it.stack, frame{childID, 0})
			continue
		}

		if f.idx < n {
			k, v, _ := readCell(p, f.idx)
			if k > it.end {
				return false
			}
			it.k, it.v, f.idx = k, v, f.idx+1
			it.subTreeDone = false
			return true
		}

		it.stack = it.stack[:len(it.stack)-1]
		it.subTreeDone = true
	}
	return false
}

func (it *RangeIterator) Key() int64    { return it.k }
func (it *RangeIterator) Value() []byte { return it.v }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }

// --- Header & Boilerplate ---

func (t *BTree) writeHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(p[:4], t.rootID)
	return t.pg.Write(1, p)
}

func (t *BTree) readHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	t.rootID = binary.LittleEndian.Uint32(p[:4])
	return nil
}

func (t *BTree) Delete(k int64) error { return nil } // To be implemented

func (t *BTree) Close() error {
	_ = t.writeHeader()
	return t.pg.Close()
}
