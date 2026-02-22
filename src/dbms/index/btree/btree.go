package btree

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

const (
	typeInternal = byte(0)
	typeLeaf     = byte(1)
	offType      = 0
	offNumKeys   = 1
	offFirstPtr  = 3
	offSlots     = 11
	slotSize     = 28
	maxKeys      = (pager.PageSize - offSlots) / slotSize
)

type BTree struct {
	pg      *pager.Pager
	valFile *os.File
	rootID  uint64
	valSize int64
}

func Open(path string, cachePages int) (*BTree, error) {
	pg, err := pager.Open(path+".bt", cachePages)
	if err != nil {
		return nil, err
	}
	vf, err := os.OpenFile(path+".bv", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	info, _ := vf.Stat()
	t := &BTree{pg: pg, valFile: vf, valSize: info.Size()}

	if pg.PageCount() <= 2 {
		_, _ = pg.Allocate() // Page 1: Header
		rootID, _ := pg.Allocate()
		t.rootID = rootID
		p := new(pager.Page)
		p[offType] = typeLeaf
		t.setNumKeys(p, 0)
		_ = pg.Write(rootID, p)
		_ = t.writeHeader()
	} else {
		_ = t.readHeader()
	}
	return t, nil
}

// ─── Search ───────────────────────────────────────────────────────────────────

func (t *BTree) Get(key int64) ([]byte, error) {
	currID := t.rootID
	for {
		p, err := t.pg.Read(currID)
		if err != nil {
			return nil, err
		}
		n := t.getNumKeys(p)
		idx := t.findIdx(p, key, n)

		if idx < n {
			k, vo, vl, _ := t.getSlot(p, idx)
			if k == key {
				return t.readValue(vo, vl)
			}
		}
		if p[offType] == typeLeaf {
			return nil, nil
		}
		currID = t.getChild(p, idx)
	}
}

// ─── Insertion ────────────────────────────────────────────────────────────────

func (t *BTree) Insert(key int64, value []byte) error {
	vOff, err := t.appendValue(value)
	if err != nil {
		return err
	}
	vLen := uint32(len(value))

	k, vo, vl, rc, split, err := t.insertRec(t.rootID, key, vOff, vLen)
	if err != nil {
		return err
	}
	if split {
		newRoot, _ := t.pg.Allocate()
		p := new(pager.Page)
		p[offType] = typeInternal
		t.setNumKeys(p, 1)
		binary.LittleEndian.PutUint64(p[offFirstPtr:offFirstPtr+8], t.rootID)
		t.putSlot(p, 0, k, vo, vl, rc)
		_ = t.pg.Write(newRoot, p)
		t.rootID = newRoot
		return t.writeHeader()
	}
	return nil
}

func (t *BTree) insertRec(id uint64, key int64, vo int64, vl uint32) (int64, int64, uint32, uint64, bool, error) {
	p, err := t.pg.Read(id)
	if err != nil {
		return 0, 0, 0, 0, false, err
	}
	n := t.getNumKeys(p)
	idx := t.findIdx(p, key, n)

	if idx < n {
		if k, _, _, rc := t.getSlot(p, idx); k == key {
			t.putSlot(p, idx, key, vo, vl, rc)
			return 0, 0, 0, 0, false, t.pg.Write(id, p)
		}
	}

	if p[offType] == typeLeaf {
		return t.handleInsert(id, p, n, idx, key, vo, vl, 0)
	}

	childID := t.getChild(p, idx)
	k, v_o, v_l, r_c, split, err := t.insertRec(childID, key, vo, vl)
	if err != nil || !split {
		return 0, 0, 0, 0, false, err
	}
	return t.handleInsert(id, p, n, idx, k, v_o, v_l, r_c)
}

type slotData struct {
	k  int64  // Key: The actual indexed value
	vo int64  // Value Offset: The byte-offset in the .bv file where the data starts.
	vl uint32 // Value Length: How many bytes to read from that offset.
	rc uint64 // Right Child: The Page ID of the child node containing keys > k.
}

func (t *BTree) handleInsert(id uint64, p *pager.Page, n, idx int, k int64, vo int64, vl uint32, rc uint64) (int64, int64, uint32, uint64, bool, error) {
	if n < maxKeys {
		for i := n; i > idx; i-- {
			t.copySlot(p, i, p, i-1)
		}
		t.putSlot(p, idx, k, vo, vl, rc)
		t.setNumKeys(p, n+1)
		return 0, 0, 0, 0, false, t.pg.Write(id, p)
	}

	tmp := make([]slotData, n+1)
	for i := 0; i < n; i++ {
		tk, tvo, tvl, trc := t.getSlot(p, i)
		if i < idx {
			tmp[i] = slotData{tk, tvo, tvl, trc}
		} else {
			tmp[i+1] = slotData{tk, tvo, tvl, trc}
		}
	}
	tmp[idx] = slotData{k, vo, vl, rc}

	mid := (n + 1) / 2
	median := tmp[mid]
	newID, _ := t.pg.Allocate()
	rightPg := new(pager.Page)
	rightPg[offType] = p[offType]

	t.setNumKeys(p, mid)
	for i := 0; i < mid; i++ {
		t.putSlot(p, i, tmp[i].k, tmp[i].vo, tmp[i].vl, tmp[i].rc)
	}

	binary.LittleEndian.PutUint64(rightPg[offFirstPtr:offFirstPtr+8], median.rc)
	t.setNumKeys(rightPg, (n+1)-(mid+1))
	for i := mid + 1; i < n+1; i++ {
		t.putSlot(rightPg, i-(mid+1), tmp[i].k, tmp[i].vo, tmp[i].vl, tmp[i].rc)
	}

	_ = t.pg.Write(id, p)
	_ = t.pg.Write(newID, rightPg)
	return median.k, median.vo, median.vl, newID, true, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (t *BTree) getSlot(p *pager.Page, i int) (int64, int64, uint32, uint64) {
	o := offSlots + i*slotSize
	return int64(binary.LittleEndian.Uint64(p[o : o+8])),
		int64(binary.LittleEndian.Uint64(p[o+8 : o+16])),
		binary.LittleEndian.Uint32(p[o+16 : o+20]),
		binary.LittleEndian.Uint64(p[o+20 : o+28])
}

func (t *BTree) putSlot(p *pager.Page, i int, k int64, vo int64, vl uint32, rc uint64) {
	o := offSlots + i*slotSize
	binary.LittleEndian.PutUint64(p[o:o+8], uint64(k))
	binary.LittleEndian.PutUint64(p[o+8:o+16], uint64(vo))
	binary.LittleEndian.PutUint32(p[o+16:o+20], vl)
	binary.LittleEndian.PutUint64(p[o+20:o+28], rc)
}

func (t *BTree) getNumKeys(p *pager.Page) int {
	return int(binary.LittleEndian.Uint16(p[offNumKeys : offNumKeys+2]))
}

func (t *BTree) setNumKeys(p *pager.Page, n int) {
	binary.LittleEndian.PutUint16(p[offNumKeys:offNumKeys+2], uint16(n))
}

func (t *BTree) copySlot(dst *pager.Page, i int, src *pager.Page, j int) {
	k, vo, vl, rc := t.getSlot(src, j)
	t.putSlot(dst, i, k, vo, vl, rc)
}

func (t *BTree) getChild(p *pager.Page, i int) uint64 {
	if i == 0 {
		return binary.LittleEndian.Uint64(p[offFirstPtr : offFirstPtr+8])
	}
	_, _, _, rc := t.getSlot(p, i-1)
	return rc
}

func (t *BTree) findIdx(p *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		m := (lo + hi) / 2
		k, _, _, _ := t.getSlot(p, m)
		if k < key {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

func (t *BTree) appendValue(v []byte) (int64, error) {
	off := t.valSize
	_, err := t.valFile.WriteAt(v, off)
	if err != nil {
		return 0, err
	}
	t.valSize += int64(len(v))
	return off, nil
}

func (t *BTree) readValue(o int64, l uint32) ([]byte, error) {
	buf := make([]byte, l)
	_, err := t.valFile.ReadAt(buf, o)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

func (t *BTree) writeHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(p[:8], t.rootID)
	return t.pg.Write(1, p)
}

func (t *BTree) readHeader() error {
	p, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	t.rootID = binary.LittleEndian.Uint64(p[:8])
	return nil
}

// Not implemented since not benchmarked
func (t *BTree) Delete(k int64) error { return nil }

func (t *BTree) Close() error {
	_ = t.writeHeader()
	_ = t.valFile.Close()
	return t.pg.Close()
}

// ─── Iterator ─────────────────────────────────────────────────────────────────

type RangeIterator struct {
	tree  *BTree
	end   int64
	stack []frame
	k     int64
	v     []byte
	err   error
}

type frame struct {
	id  uint64
	idx int
}

func (t *BTree) Range(start, end int64) (index.Iterator, error) {
	it := &RangeIterator{tree: t, end: end}
	curr := t.rootID
	for {
		p, err := t.pg.Read(curr)
		if err != nil {
			return nil, err
		}
		idx := t.findIdx(p, start, t.getNumKeys(p))
		it.stack = append(it.stack, frame{curr, idx})
		if p[offType] == typeLeaf {
			break
		}
		curr = t.getChild(p, idx)
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
		n := it.tree.getNumKeys(p)

		if f.idx < n {
			k, vo, vl, rc := it.tree.getSlot(p, f.idx)
			if k > it.end {
				return false
			}
			it.k = k
			it.v, it.err = it.tree.readValue(vo, vl)
			f.idx++
			if p[offType] == typeInternal {
				it.stack = append(it.stack, frame{rc, 0})
			}
			return true
		}
		it.stack = it.stack[:len(it.stack)-1]
	}
	return false
}

func (it *RangeIterator) Key() int64    { return it.k }
func (it *RangeIterator) Value() []byte { return it.v }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
