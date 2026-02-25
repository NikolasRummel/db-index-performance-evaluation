package shared

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/index/btpage"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

type NodeAccessor interface {
	CellSize(isLeaf bool, value []byte) int

	ReadCell(p *pager.Page, i int, isLeaf bool) (key int64, value []byte, leftChild uint32)

	WriteCell(p *pager.Page, off int, key int64, value []byte, leftChild uint32, isLeaf bool)

	OverwriteValue(p *pager.Page, i int, newVal []byte, isLeaf bool) // ERROR IF NEW VAL DOES NOT FIT IN OLD SPACE!!

	CopyUpLeaves() bool

	LinkLeaves(left, right *pager.Page, newRightID uint32, oldNext uint32)
}

type Tree struct {
	Pg     *pager.Pager
	RootID uint32
	Acc    NodeAccessor
}

// ─── helpers ───────────────────────────────────

func isLeaf(p *pager.Page) bool { return p[btpage.OffType] == btpage.TypeLeaf }

func (t *Tree) readCell(p *pager.Page, i int) (int64, []byte, uint32) {
	return t.Acc.ReadCell(p, i, isLeaf(p))
}

func (t *Tree) cellSize(p *pager.Page, value []byte) int {
	return t.Acc.CellSize(isLeaf(p), value)
}

func (t *Tree) AppendCell(p *pager.Page, key int64, value []byte, leftChild uint32) {
	n := btpage.NumCells(p)
	off := btpage.AllocCell(p, t.Acc.CellSize(isLeaf(p), value))
	t.Acc.WriteCell(p, off, key, value, leftChild, isLeaf(p))
	btpage.SetCellPtr(p, n, uint16(off))
	btpage.SetNumCells(p, n+1)
}

func DeleteCell(p *pager.Page, i int) {
	n := btpage.NumCells(p)
	for j := i; j < n-1; j++ {
		btpage.SetCellPtr(p, j, btpage.CellPtr(p, j+1))
	}
	btpage.SetNumCells(p, n-1)
}

func ChildAt(p *pager.Page, idx, n int, acc NodeAccessor) uint32 {
	if idx == n {
		return btpage.Rightmost(p)
	}
	_, _, lc := acc.ReadCell(p, idx, false)
	return lc
}

func FindIdx(p *pager.Page, key int64, n int, acc NodeAccessor, leaf bool) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _, _ := acc.ReadCell(p, m, leaf)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

func (t *Tree) Get(key int64) ([]byte, error) {
	curr := uint64(t.RootID)
	for {
		p, err := t.Pg.Read(curr)
		if err != nil {
			return nil, err
		}
		n := btpage.NumCells(p)
		leaf := isLeaf(p)
		idx := FindIdx(p, key, n, t.Acc, leaf)
		if idx < n {
			k, val, _ := t.Acc.ReadCell(p, idx, leaf)
			if k == key && val != nil {
				return val, nil
			}
		}
		if leaf {
			return nil, nil
		}
		curr = uint64(ChildAt(p, idx, n, t.Acc))
	}
}

func (t *Tree) Insert(key int64, value []byte) error {
	mk, mv, rightID, split, err := t.insertRec(uint64(t.RootID), key, value)
	if err != nil {
		return err
	}
	if !split {
		return nil
	}
	newRoot, _ := t.Pg.Allocate()
	p := new(pager.Page)
	btpage.InitPage(p, btpage.TypeInternal)
	btpage.SetRightmost(p, uint32(rightID))
	t.AppendCell(p, mk, mv, t.RootID)
	_ = t.Pg.Write(newRoot, p)
	t.RootID = uint32(newRoot)
	return t.WriteHeader()
}

func (t *Tree) insertRec(id uint64, key int64, value []byte) (int64, []byte, uint64, bool, error) {
	p, err := t.Pg.Read(id)
	if err != nil {
		return 0, nil, 0, false, err
	}
	n := btpage.NumCells(p)
	leaf := isLeaf(p)
	idx := FindIdx(p, key, n, t.Acc, leaf)

	// Handle existing key: overwrite in-place if value fits, else delete+reinsert.
	if idx < n {
		if k, oldVal, _ := t.Acc.ReadCell(p, idx, leaf); k == key {
			if len(value) <= len(oldVal) {
				t.Acc.OverwriteValue(p, idx, value, leaf)
				return 0, nil, 0, false, t.Pg.Write(id, p)
			}
			DeleteCell(p, idx)
			n--
		}
	}

	if leaf {
		return t.doInsert(id, p, n, idx, key, value, 0)
	}

	// Recurse into child.
	childID := uint64(ChildAt(p, idx, n, t.Acc))
	mk, mv, rc, split, err := t.insertRec(childID, key, value)
	if err != nil || !split {
		return 0, nil, 0, false, err
	}

	// Re-read after child write (page may have been evicted from cache).
	p, err = t.Pg.Read(id)
	if err != nil {
		return 0, nil, 0, false, err
	}
	n = btpage.NumCells(p)
	idx = FindIdx(p, mk, n, t.Acc, false)
	return t.doInsert(id, p, n, idx, mk, mv, rc)
}

type CellData struct {
	Key       int64
	Value     []byte
	LeftChild uint32
}

func (t *Tree) doInsert(id uint64, p *pager.Page, n, idx int, key int64, value []byte, rightChild uint64) (int64, []byte, uint64, bool, error) {
	leaf := isLeaf(p)

	if btpage.FreeSpace(p, n) >= t.Acc.CellSize(leaf, value) {
		for i := n; i > idx; i-- {
			btpage.SetCellPtr(p, i, btpage.CellPtr(p, i-1))
		}
		off := btpage.AllocCell(p, t.Acc.CellSize(leaf, value))
		t.Acc.WriteCell(p, off, key, value, ChildAt(p, idx, n, t.Acc), leaf)
		btpage.SetCellPtr(p, idx, uint16(off))

		if idx == n {
			btpage.SetRightmost(p, uint32(rightChild))
		} else if !leaf {
			off1 := int(btpage.CellPtr(p, idx+1))
			binary.LittleEndian.PutUint32(p[off1:off1+4], uint32(rightChild))
		}
		btpage.SetNumCells(p, n+1)
		return 0, nil, 0, false, t.Pg.Write(id, p)
	}

	return t.splitNode(id, p, n, idx, key, value, rightChild)
}

func (t *Tree) splitNode(id uint64, p *pager.Page, n, idx int, key int64, value []byte, rightChild uint64) (int64, []byte, uint64, bool, error) {
	leaf := isLeaf(p)
	pageType := p[btpage.OffType] // cache before InitPage zeroes it

	all := make([]CellData, n+1)
	for i := 0; i < n; i++ {
		k, v, lc := t.Acc.ReadCell(p, i, leaf)
		all[i] = CellData{k, v, lc}
	}
	copy(all[idx+1:], all[idx:n])
	all[idx] = CellData{key, value, ChildAt(p, idx, n, t.Acc)}

	oldRightmost := btpage.Rightmost(p)
	if idx == n {
		oldRightmost = uint32(rightChild)
	} else if !leaf {
		all[idx+1].LeftChild = uint32(rightChild)
	}

	mid := (n + 1) / 2
	median := all[mid]

	newID, _ := t.Pg.Allocate()
	right := new(pager.Page)

	btpage.InitPage(p, pageType)
	if leaf {
		for i := 0; i < mid; i++ {
			t.AppendCell(p, all[i].Key, all[i].Value, all[i].LeftChild)
		}
	} else {
		btpage.SetRightmost(p, median.LeftChild)
		for i := 0; i < mid; i++ {
			t.AppendCell(p, all[i].Key, all[i].Value, all[i].LeftChild)
		}
	}

	btpage.InitPage(right, pageType)
	if leaf {
		// B+: copy-up since leave will always contain value. For normal B trees, mid key goes to parent and removed from child.
		start := mid
		if !t.Acc.CopyUpLeaves() {
			start = mid + 1
		}
		for i := start; i <= n; i++ {
			t.AppendCell(right, all[i].Key, all[i].Value, all[i].LeftChild)
		}
		t.Acc.LinkLeaves(p, right, uint32(newID), oldRightmost) // For B tree not implemented since no linked list here
	} else {
		btpage.SetRightmost(right, oldRightmost)
		for i := mid + 1; i <= n; i++ {
			t.AppendCell(right, all[i].Key, all[i].Value, all[i].LeftChild)
		}
	}

	_ = t.Pg.Write(id, p)
	_ = t.Pg.Write(newID, right)
	return median.Key, median.Value, newID, true, nil
}

func (t *Tree) FindLeaf(key int64) (uint64, error) {
	curr := uint64(t.RootID)
	for {
		p, err := t.Pg.Read(curr)
		if err != nil {
			return 0, err
		}
		if isLeaf(p) {
			return curr, nil
		}
		n := btpage.NumCells(p)
		idx := FindIdx(p, key, n, t.Acc, false)
		curr = uint64(ChildAt(p, idx, n, t.Acc))
	}
}

func (t *Tree) WriteHeader() error {
	p, err := t.Pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(p[:4], t.RootID)
	return t.Pg.Write(1, p)
}

func (t *Tree) ReadHeader() error {
	p, err := t.Pg.Read(1)
	if err != nil {
		return err
	}
	t.RootID = binary.LittleEndian.Uint32(p[:4])
	return nil
}
