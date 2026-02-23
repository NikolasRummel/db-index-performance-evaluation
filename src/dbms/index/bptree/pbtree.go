package bptree

import (
	"encoding/binary"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const (
	order   = 200
	maxKeys = order - 1

	typeInternal = byte(0)
	typeLeaf     = byte(1)

	offType     = 0
	offNumKeys  = 1
	offNextLeaf = 3  // Leaf only
	offPrevLeaf = 11 // Leaf only
	offFirstPtr = 3  // Internal only
	offSlots    = 19
	slotSize    = 20
)

type BPTree struct {
	pg      *pager.Pager
	valFile *os.File
	rootID  uint64
	valSize int64
}

func Open(path string, cachePages int) (*BPTree, error) {
	pg, err := pager.Open(path+".bpt", cachePages)
	if err != nil {
		return nil, err
	}

	vf, err := os.OpenFile(path+".bpv", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		pg.Close()
		return nil, err
	}

	info, _ := vf.Stat()
	t := &BPTree{pg: pg, valFile: vf, valSize: info.Size()}

	if pg.PageCount() <= 2 {
		_, _ = pg.Allocate() // Header
		rootID, _ := pg.Allocate()
		t.rootID = rootID
		_ = t.initLeaf(rootID, pager.InvalidPage, pager.InvalidPage)
		_ = t.writeHeader()
	} else {
		_ = t.readHeader()
	}
	return t, nil
}

func (t *BPTree) Insert(key int64, value []byte) error {
	offset, err := t.appendValue(value)
	if err != nil {
		return err
	}

	midKey, newPageID, split, err := t.insertNode(t.rootID, key, offset, uint32(len(value)))
	if err != nil {
		return err
	}

	if split {
		newRoot, _ := t.pg.Allocate()
		if err := t.initInternal(newRoot, t.rootID, midKey, newPageID); err != nil {
			return err
		}
		t.rootID = newRoot
		return t.writeHeader()
	}
	return nil
}

func (t *BPTree) Get(key int64) ([]byte, error) {
	leafID, err := t.findLeaf(key)
	if err != nil {
		return nil, err
	}
	pg, _ := t.pg.Read(leafID)
	n := numKeys(pg)
	idx := findKeyIndex(pg, key, n)
	if idx < n {
		if k, valOffset, valLen := getSlot(pg, idx); k == key {
			return t.readValue(int64(valOffset), valLen)
		}
	}
	return nil, nil
}

// Not implemented since not benchmarked
func (t *BPTree) Delete(key int64) error { return nil }

// ─── Insertion & Splitting ───────────────────────────────────────────────────

func (t *BPTree) insertNode(nodeID uint64, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
	pg, _ := t.pg.Read(nodeID)
	n := numKeys(pg)

	if pg[offType] == typeLeaf {
		idx := findKeyIndex(pg, key, n)
		if idx < n {
			if k, _, _ := getSlot(pg, idx); k == key {
				putSlot(pg, idx, key, uint64(valOffset), valLen)
				return 0, 0, false, t.pg.Write(nodeID, pg)
			}
		}
		for i := n; i > idx; i-- {
			k, p, l := getSlot(pg, i-1)
			putSlot(pg, i, k, p, l)
		}
		putSlot(pg, idx, key, uint64(valOffset), valLen)
		n++
		setNumKeys(pg, n)

		if n <= maxKeys {
			return 0, 0, false, t.pg.Write(nodeID, pg)
		}
		return t.splitLeaf(nodeID, pg, n)
	}

	// Internal logic
	idx := findKeyIndex(pg, key, n)
	var childID uint64
	if idx == 0 {
		childID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	} else {
		_, childID, _ = getSlot(pg, idx-1)
	}

	midKey, newChildID, split, err := t.insertNode(childID, key, valOffset, valLen)
	if err != nil || !split {
		return 0, 0, false, err
	}

	for i := n; i > idx; i-- {
		k, p, l := getSlot(pg, i-1)
		putSlot(pg, i, k, p, l)
	}
	putSlot(pg, idx, midKey, newChildID, 0)
	n++
	setNumKeys(pg, n)

	if n <= maxKeys {
		return 0, 0, false, t.pg.Write(nodeID, pg)
	}
	return t.splitInternal(nodeID, pg, n)
}

func (t *BPTree) splitLeaf(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, _ := t.pg.Allocate()
	newPg := new(pager.Page)
	newPg[offType] = typeLeaf

	oldNext := binary.LittleEndian.Uint64(pg[offNextLeaf : offNextLeaf+8])
	binary.LittleEndian.PutUint64(newPg[offNextLeaf:], oldNext)
	binary.LittleEndian.PutUint64(newPg[offPrevLeaf:], nodeID)
	binary.LittleEndian.PutUint64(pg[offNextLeaf:], newID)

	if oldNext != pager.InvalidPage {
		nextPg, _ := t.pg.Read(oldNext)
		binary.LittleEndian.PutUint64(nextPg[offPrevLeaf:], newID)
		_ = t.pg.Write(oldNext, nextPg)
	}

	for i := mid; i < n; i++ {
		k, p, l := getSlot(pg, i)
		putSlot(newPg, i-mid, k, p, l)
	}
	setNumKeys(newPg, n-mid)
	setNumKeys(pg, mid)

	_ = t.pg.Write(nodeID, pg)
	_ = t.pg.Write(newID, newPg)
	midKey, _, _ := getSlot(newPg, 0)
	return midKey, newID, true, nil
}

func (t *BPTree) splitInternal(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, _ := t.pg.Allocate()
	newPg := new(pager.Page)
	newPg[offType] = typeInternal

	midKey, midRightChild, _ := getSlot(pg, mid)
	binary.LittleEndian.PutUint64(newPg[offFirstPtr:], midRightChild)
	for i := mid + 1; i < n; i++ {
		k, p, l := getSlot(pg, i)
		putSlot(newPg, i-(mid+1), k, p, l)
	}
	setNumKeys(newPg, n-(mid+1))
	setNumKeys(pg, mid)

	_ = t.pg.Write(nodeID, pg)
	_ = t.pg.Write(newID, newPg)
	return midKey, newID, true, nil
}

// ─── Helpers (Kept for Core Functionality) ───────────────────────────────────

func (t *BPTree) initLeaf(id, next, prev uint64) error {
	pg := new(pager.Page)
	pg[offType] = typeLeaf
	setNumKeys(pg, 0)
	binary.LittleEndian.PutUint64(pg[offNextLeaf:], next)
	binary.LittleEndian.PutUint64(pg[offPrevLeaf:], prev)
	return t.pg.Write(id, pg)
}

func (t *BPTree) initInternal(id, leftChild uint64, key int64, rightChild uint64) error {
	pg := new(pager.Page)
	pg[offType] = typeInternal
	setNumKeys(pg, 1)
	binary.LittleEndian.PutUint64(pg[offFirstPtr:], leftChild)
	putSlot(pg, 0, key, rightChild, 0)
	return t.pg.Write(id, pg)
}

func numKeys(pg *pager.Page) int {
	return int(binary.LittleEndian.Uint16(pg[offNumKeys : offNumKeys+2]))
}
func setNumKeys(pg *pager.Page, n int) {
	binary.LittleEndian.PutUint16(pg[offNumKeys:offNumKeys+2], uint16(n))
}
func getSlot(pg *pager.Page, i int) (int64, uint64, uint32) {
	o := offSlots + i*slotSize
	return int64(binary.LittleEndian.Uint64(pg[o : o+8])), binary.LittleEndian.Uint64(pg[o+8 : o+16]), binary.LittleEndian.Uint32(pg[o+16 : o+20])
}
func putSlot(pg *pager.Page, i int, key int64, ptr uint64, vlen uint32) {
	o := offSlots + i*slotSize
	binary.LittleEndian.PutUint64(pg[o:], uint64(key))
	binary.LittleEndian.PutUint64(pg[o+8:], ptr)
	binary.LittleEndian.PutUint32(pg[o+16:], vlen)
}

func findKeyIndex(pg *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _, _ := getSlot(pg, m)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

func (t *BPTree) findLeaf(key int64) (uint64, error) {
	id := t.rootID
	for {
		pg, _ := t.pg.Read(id)
		if pg[offType] == typeLeaf {
			return id, nil
		}
		n := numKeys(pg)
		idx := findKeyIndex(pg, key, n)
		if idx == 0 {
			id = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		} else {
			_, id, _ = getSlot(pg, idx-1)
		}
	}
}

func (t *BPTree) appendValue(v []byte) (int64, error) {
	off := t.valSize
	_, err := t.valFile.WriteAt(v, off)
	t.valSize += int64(len(v))
	return off, err
}

func (t *BPTree) readValue(o int64, l uint32) ([]byte, error) {
	buf := make([]byte, l)
	_, _ = t.valFile.ReadAt(buf, o)
	return buf, nil
}

func (t *BPTree) writeHeader() error {
	p, _ := t.pg.Read(1)
	binary.LittleEndian.PutUint64(p[:8], t.rootID)
	return t.pg.Write(1, p)
}

func (t *BPTree) readHeader() error {
	p, _ := t.pg.Read(1)
	t.rootID = binary.LittleEndian.Uint64(p[:8])
	return nil
}

func (t *BPTree) Close() error { _ = t.writeHeader(); t.valFile.Close(); return t.pg.Close() }

// ─── Range Iterator ───────────────────────────────────────────────────────────

type RangeIterator struct {
	tree   *BPTree
	end    int64
	leafID uint64
	idx    int
	key    int64
	val    []byte
	err    error
	done   bool
}

func (t *BPTree) Range(start, end int64) (index.Iterator, error) {
	id, _ := t.findLeaf(start)
	pg, _ := t.pg.Read(id)
	return &RangeIterator{tree: t, end: end, leafID: id, idx: findKeyIndex(pg, start, numKeys(pg))}, nil
}

func (it *RangeIterator) Next() bool {
	if it.done || it.leafID == pager.InvalidPage {
		return false
	}
	for {
		pg, _ := it.tree.pg.Read(it.leafID)
		n := numKeys(pg)
		if it.idx < n {
			k, vo, vl := getSlot(pg, it.idx)
			if k > it.end {
				it.done = true
				return false
			}
			it.idx++
			it.key = k
			it.val, _ = it.tree.readValue(int64(vo), vl)
			return true
		}
		it.leafID = binary.LittleEndian.Uint64(pg[offNextLeaf : offNextLeaf+8])
		it.idx = 0
		if it.leafID == pager.InvalidPage {
			it.done = true
			return false
		}
	}
}

func (it *RangeIterator) Key() int64    { return it.key }
func (it *RangeIterator) Value() []byte { return it.val }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
