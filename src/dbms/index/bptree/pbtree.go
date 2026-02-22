// Package bptree implements a disk-based B+ tree using the pager package.
//
// The key difference from a B-tree:
//   - ALL data (key+value pointer) lives in leaf nodes.
//   - Internal nodes hold only separator keys + child pointers (no values).
//   - Leaf nodes are linked in a doubly-linked list for O(1) range scan advancement.
//
// Leaf page layout (4096 bytes):
//
//	[0]     uint8  — node type (always 1)
//	[1..2]  uint16 — number of keys
//	[3..10] uint64 — page ID of next leaf (InvalidPage if last)
//	[11..18] uint64 — page ID of prev leaf (InvalidPage if first)
//	[19..]  slots: [key int64][valOffset uint64][valLen uint32] = 20 bytes each
//
// Internal page layout (4096 bytes):
//
//	[0]     uint8  — node type (always 0)
//	[1..2]  uint16 — number of keys
//	[3..10] uint64 — first child page ID
//	[11..]  slots: [key int64][rightChild uint64][_pad uint32] = 20 bytes each
package bptree

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	order   = 200
	maxKeys = order - 1
	minKeys = (order - 1) / 2

	typeInternal = byte(0)
	typeLeaf     = byte(1)

	offType     = 0
	offNumKeys  = 1  // uint16
	offNextLeaf = 3  // uint64 — linked list pointer (leaf only)
	offPrevLeaf = 11 // uint64 — linked list pointer (leaf only)
	offFirstPtr = 3  // uint64 — first child (internal only, same offset as offNextLeaf)
	offSlots    = 19 // start of slots (leaf: after prev ptr; internal: after firstPtr + 8 padding)

	slotSize = 20 // [key int64 8B][ptr/offset uint64 8B][len uint32 4B]
)

// ─── BPTree ───────────────────────────────────────────────────────────────────

// BPTree is a disk-based B+ tree.
type BPTree struct {
	pg      *pager.Pager
	valFile *os.File
	rootID  uint64
	valSize int64
}

// Open opens (or creates) a B+ tree at the given base path.
// cachePages controls the LRU page cache size (e.g. 256 = 1MB, 4096 = 16MB).
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
	info, err := vf.Stat()
	if err != nil {
		return nil, err
	}

	t := &BPTree{pg: pg, valFile: vf, valSize: info.Size()}

	if pg.PageCount() <= 2 {
		_, err = pg.Allocate() // page 1: bptree header
		if err != nil {
			return nil, err
		}
		rootID, err := pg.Allocate() // page 2: initial root leaf
		if err != nil {
			return nil, err
		}
		t.rootID = rootID
		if err := t.initLeaf(rootID, pager.InvalidPage, pager.InvalidPage); err != nil {
			return nil, err
		}
		if err := t.writeHeader(); err != nil {
			return nil, err
		}
	} else {
		if err := t.readHeader(); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// Close flushes and closes all files.
func (t *BPTree) Close() error {
	if err := t.writeHeader(); err != nil {
		return err
	}
	if err := t.valFile.Close(); err != nil {
		return err
	}
	return t.pg.Close()
}

// ─── Public API ───────────────────────────────────────────────────────────────

// Insert inserts or updates the value for key.
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
		newRoot, err := t.pg.Allocate()
		if err != nil {
			return err
		}
		if err := t.initInternal(newRoot, t.rootID, midKey, newPageID); err != nil {
			return err
		}
		t.rootID = newRoot
		if err := t.writeHeader(); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves the value for key, or nil if not found.
func (t *BPTree) Get(key int64) ([]byte, error) {
	leafID, err := t.findLeaf(key)
	if err != nil {
		return nil, err
	}
	pg, err := t.pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	n := numKeys(pg)
	idx := findKeyIndex(pg, key, n)
	if idx < n {
		if k, valOffset, valLen := getSlot(pg, idx); k == key {
			return t.readValue(int64(valOffset), valLen)
		}
	}
	return nil, nil
}

// Delete removes the key from the tree.
func (t *BPTree) Delete(key int64) error {
	found, err := t.deleteNode(t.rootID, key, pager.InvalidPage, 0)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	// Collapse empty root.
	pg, err := t.pg.Read(t.rootID)
	if err != nil {
		return err
	}
	if numKeys(pg) == 0 && pg[offType] == typeInternal {
		t.rootID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		return t.writeHeader()
	}
	return nil
}

// Range returns an iterator over keys in [start, end] inclusive.
// B+ trees shine here: we find the leaf once, then follow next-leaf pointers.
func (t *BPTree) Range(start, end int64) (index.Iterator, error) {
	leafID, err := t.findLeaf(start)
	if err != nil {
		return nil, err
	}
	pg, err := t.pg.Read(leafID)
	if err != nil {
		return nil, err
	}
	n := numKeys(pg)
	startIdx := findKeyIndex(pg, start, n)
	it := &RangeIterator{
		tree:   t,
		end:    end,
		leafID: leafID,
		idx:    startIdx,
	}
	return it, nil
}

// ─── Node helpers ─────────────────────────────────────────────────────────────

func (t *BPTree) initLeaf(id, next, prev uint64) error {
	pg := new(pager.Page)
	pg[offType] = typeLeaf
	binary.LittleEndian.PutUint16(pg[offNumKeys:], 0)
	binary.LittleEndian.PutUint64(pg[offNextLeaf:], next)
	binary.LittleEndian.PutUint64(pg[offPrevLeaf:], prev)
	return t.pg.Write(id, pg)
}

func (t *BPTree) initInternal(id, leftChild uint64, key int64, rightChild uint64) error {
	pg := new(pager.Page)
	pg[offType] = typeInternal
	binary.LittleEndian.PutUint16(pg[offNumKeys:], 1)
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

func slotOffset(i int) int {
	return offSlots + i*slotSize
}

func getSlot(pg *pager.Page, i int) (key int64, ptr uint64, vlen uint32) {
	off := slotOffset(i)
	key = int64(binary.LittleEndian.Uint64(pg[off : off+8]))
	ptr = binary.LittleEndian.Uint64(pg[off+8 : off+16])
	vlen = binary.LittleEndian.Uint32(pg[off+16 : off+20])
	return
}

func putSlot(pg *pager.Page, i int, key int64, ptr uint64, vlen uint32) {
	off := slotOffset(i)
	binary.LittleEndian.PutUint64(pg[off:], uint64(key))
	binary.LittleEndian.PutUint64(pg[off+8:], ptr)
	binary.LittleEndian.PutUint32(pg[off+16:], vlen)
}

func findKeyIndex(pg *pager.Page, key int64, n int) int {
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		k, _, _ := getSlot(pg, mid)
		if k < key {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// findLeaf descends to the leaf page that should contain key.
func (t *BPTree) findLeaf(key int64) (uint64, error) {
	nodeID := t.rootID
	for {
		pg, err := t.pg.Read(nodeID)
		if err != nil {
			return 0, err
		}
		if pg[offType] == typeLeaf {
			return nodeID, nil
		}
		n := numKeys(pg)
		idx := findKeyIndex(pg, key, n)
		if idx == 0 {
			nodeID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		} else {
			_, nodeID, _ = getSlot(pg, idx-1)
		}
	}
}

// ─── Insert ───────────────────────────────────────────────────────────────────

func (t *BPTree) insertNode(nodeID uint64, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
	pg, err := t.pg.Read(nodeID)
	if err != nil {
		return 0, 0, false, err
	}
	n := numKeys(pg)

	if pg[offType] == typeLeaf {
		return t.insertLeaf(nodeID, pg, n, key, valOffset, valLen)
	}
	return t.insertInternal(nodeID, pg, n, key, valOffset, valLen)
}

func (t *BPTree) insertLeaf(nodeID uint64, pg *pager.Page, n int, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
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

func (t *BPTree) splitLeaf(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, err := t.pg.Allocate()
	if err != nil {
		return 0, 0, false, err
	}
	newPg := new(pager.Page)
	newPg[offType] = typeLeaf

	// Fix up the linked list: newID goes between nodeID and nodeID's old next.
	oldNext := binary.LittleEndian.Uint64(pg[offNextLeaf : offNextLeaf+8])
	binary.LittleEndian.PutUint64(newPg[offNextLeaf:], oldNext)
	binary.LittleEndian.PutUint64(newPg[offPrevLeaf:], nodeID)
	binary.LittleEndian.PutUint64(pg[offNextLeaf:], newID)

	// Update old next's prev pointer.
	if oldNext != pager.InvalidPage {
		nextPg, err := t.pg.Read(oldNext)
		if err != nil {
			return 0, 0, false, err
		}
		binary.LittleEndian.PutUint64(nextPg[offPrevLeaf:], newID)
		if err := t.pg.Write(oldNext, nextPg); err != nil {
			return 0, 0, false, err
		}
	}

	// Copy right half to new leaf. B+ trees keep ALL keys in leaves.
	for i := mid; i < n; i++ {
		k, p, l := getSlot(pg, i)
		putSlot(newPg, i-mid, k, p, l)
	}
	setNumKeys(newPg, n-mid)
	setNumKeys(pg, mid)

	if err := t.pg.Write(nodeID, pg); err != nil {
		return 0, 0, false, err
	}
	if err := t.pg.Write(newID, newPg); err != nil {
		return 0, 0, false, err
	}

	// The separator key pushed up is the first key of the new right leaf.
	midKey, _, _ := getSlot(newPg, 0)
	return midKey, newID, true, nil
}

func (t *BPTree) insertInternal(nodeID uint64, pg *pager.Page, n int, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
	idx := findKeyIndex(pg, key, n)
	var childID uint64
	if idx == 0 {
		childID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	} else {
		_, childID, _ = getSlot(pg, idx-1)
	}

	midKey, newChildID, split, err := t.insertNode(childID, key, valOffset, valLen)
	if err != nil {
		return 0, 0, false, err
	}
	if !split {
		return 0, 0, false, nil
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

func (t *BPTree) splitInternal(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, err := t.pg.Allocate()
	if err != nil {
		return 0, 0, false, err
	}
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

	if err := t.pg.Write(nodeID, pg); err != nil {
		return 0, 0, false, err
	}
	if err := t.pg.Write(newID, newPg); err != nil {
		return 0, 0, false, err
	}
	return midKey, newID, true, nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// childPageID returns the page ID of the child at position pos.
// pos=0 → firstPtr, pos=i → right-child ptr of slot i-1.
func childPageID(pg *pager.Page, pos int) uint64 {
	if pos == 0 {
		return binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	}
	_, id, _ := getSlot(pg, pos-1)
	return id
}

func (t *BPTree) deleteNode(nodeID uint64, key int64, parentID uint64, parentIdx int) (bool, error) {
	pg, err := t.pg.Read(nodeID)
	if err != nil {
		return false, err
	}
	n := numKeys(pg)

	if pg[offType] == typeLeaf {
		return t.deleteFromLeaf(nodeID, pg, n, key)
	}

	childPos := findKeyIndex(pg, key, n)
	var childID uint64
	if childPos == 0 {
		childID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	} else {
		_, childID, _ = getSlot(pg, childPos-1)
	}

	found, err := t.deleteNode(childID, key, nodeID, childPos)
	if err != nil || !found {
		return found, err
	}

	childPg, err := t.pg.Read(childID)
	if err != nil {
		return true, err
	}
	if numKeys(childPg) >= minKeys {
		return true, nil
	}

	// Re-read parent — recursive writes may have changed it.
	parentPg, err := t.pg.Read(nodeID)
	if err != nil {
		return true, err
	}
	parentN := numKeys(parentPg)
	return true, t.rebalance(nodeID, parentPg, parentN, childPos, childID)
}

func (t *BPTree) deleteFromLeaf(nodeID uint64, pg *pager.Page, n int, key int64) (bool, error) {
	idx := findKeyIndex(pg, key, n)
	if idx >= n {
		return false, nil
	}
	k, _, _ := getSlot(pg, idx)
	if k != key {
		return false, nil
	}
	for i := idx; i < n-1; i++ {
		k2, p, l := getSlot(pg, i+1)
		putSlot(pg, i, k2, p, l)
	}
	setNumKeys(pg, n-1)
	return true, t.pg.Write(nodeID, pg)
}

// rebalance fixes an underflowing child at childPos.
// Borrows from a sibling if possible; merges only as last resort.
func (t *BPTree) rebalance(parentID uint64, parentPg *pager.Page, parentN int, childPos int, childID uint64) error {
	// Try right sibling first.
	if childPos < parentN {
		rightID := childPageID(parentPg, childPos+1)
		rightPg, err := t.pg.Read(rightID)
		if err != nil {
			return err
		}
		if numKeys(rightPg) > minKeys {
			return t.borrowFromRight(parentID, parentPg, childPos, childID, rightID, rightPg)
		}
		return t.mergeNodes(parentID, parentPg, parentN, childPos, childID, rightID)
	}

	// Try left sibling.
	leftID := childPageID(parentPg, childPos-1)
	leftPg, err := t.pg.Read(leftID)
	if err != nil {
		return err
	}
	if numKeys(leftPg) > minKeys {
		return t.borrowFromLeft(parentID, parentPg, childPos-1, leftID, leftPg, childID)
	}
	return t.mergeNodes(parentID, parentPg, parentN, childPos-1, leftID, childID)
}

// borrowFromRight rotates the right sibling's first key through the parent into the child.
func (t *BPTree) borrowFromRight(parentID uint64, parentPg *pager.Page, sepIdx int, childID, rightID uint64, rightPg *pager.Page) error {
	childPg, err := t.pg.Read(childID)
	if err != nil {
		return err
	}
	cn := numKeys(childPg)
	rn := numKeys(rightPg)
	sepKey, _, _ := getSlot(parentPg, sepIdx)

	if childPg[offType] == typeInternal {
		rightFirstChild := binary.LittleEndian.Uint64(rightPg[offFirstPtr : offFirstPtr+8])
		putSlot(childPg, cn, sepKey, rightFirstChild, 0)
		setNumKeys(childPg, cn+1)
		newSepKey, newFirstChild, _ := getSlot(rightPg, 0)
		binary.LittleEndian.PutUint64(rightPg[offFirstPtr:], newFirstChild)
		for i := 0; i < rn-1; i++ {
			k, p, l := getSlot(rightPg, i+1)
			putSlot(rightPg, i, k, p, l)
		}
		setNumKeys(rightPg, rn-1)
		putSlot(parentPg, sepIdx, newSepKey, rightID, 0)
	} else {
		// Leaf: move right's first entry to end of child.
		newKey, newPtr, newLen := getSlot(rightPg, 0)
		putSlot(childPg, cn, newKey, newPtr, newLen)
		setNumKeys(childPg, cn+1)
		for i := 0; i < rn-1; i++ {
			k, p, l := getSlot(rightPg, i+1)
			putSlot(rightPg, i, k, p, l)
		}
		setNumKeys(rightPg, rn-1)
		// B+ tree: separator = right's new first key (keys stay in leaves).
		newSepKey, _, _ := getSlot(rightPg, 0)
		putSlot(parentPg, sepIdx, newSepKey, rightID, 0)
	}

	if err := t.pg.Write(childID, childPg); err != nil {
		return err
	}
	if err := t.pg.Write(rightID, rightPg); err != nil {
		return err
	}
	return t.pg.Write(parentID, parentPg)
}

// borrowFromLeft rotates the left sibling's last key through the parent into the child.
func (t *BPTree) borrowFromLeft(parentID uint64, parentPg *pager.Page, sepIdx int, leftID uint64, leftPg *pager.Page, childID uint64) error {
	childPg, err := t.pg.Read(childID)
	if err != nil {
		return err
	}
	cn := numKeys(childPg)
	ln := numKeys(leftPg)
	sepKey, _, _ := getSlot(parentPg, sepIdx)

	if childPg[offType] == typeInternal {
		// Shift child right, prepend separator.
		oldFirstChild := binary.LittleEndian.Uint64(childPg[offFirstPtr : offFirstPtr+8])
		for i := cn; i > 0; i-- {
			k, p, l := getSlot(childPg, i-1)
			putSlot(childPg, i, k, p, l)
		}
		lastKey, lastChild, _ := getSlot(leftPg, ln-1)
		putSlot(childPg, 0, sepKey, oldFirstChild, 0)
		binary.LittleEndian.PutUint64(childPg[offFirstPtr:], lastChild)
		setNumKeys(childPg, cn+1)
		setNumKeys(leftPg, ln-1)
		putSlot(parentPg, sepIdx, lastKey, childID, 0)
	} else {
		// Leaf: shift child right, prepend left's last entry.
		for i := cn; i > 0; i-- {
			k, p, l := getSlot(childPg, i-1)
			putSlot(childPg, i, k, p, l)
		}
		lastKey, lastPtr, lastLen := getSlot(leftPg, ln-1)
		putSlot(childPg, 0, lastKey, lastPtr, lastLen)
		setNumKeys(childPg, cn+1)
		setNumKeys(leftPg, ln-1)
		// B+ tree: separator = child's new first key.
		putSlot(parentPg, sepIdx, lastKey, childID, 0)
	}

	if err := t.pg.Write(leftID, leftPg); err != nil {
		return err
	}
	if err := t.pg.Write(childID, childPg); err != nil {
		return err
	}
	return t.pg.Write(parentID, parentPg)
}

// mergeNodes merges right into left. For leaves, fixes the linked list.
// Only called when left.n + right.n (+ separator for internal) <= maxKeys.
func (t *BPTree) mergeNodes(parentID uint64, parentPg *pager.Page, parentN int, sepIdx int, leftID, rightID uint64) error {
	leftPg, err := t.pg.Read(leftID)
	if err != nil {
		return err
	}
	rightPg, err := t.pg.Read(rightID)
	if err != nil {
		return err
	}
	ln := numKeys(leftPg)
	rn := numKeys(rightPg)

	if leftPg[offType] == typeLeaf {
		// Concatenate + fix linked list (no separator pulled down for B+ leaves).
		for i := 0; i < rn; i++ {
			k, p, l := getSlot(rightPg, i)
			putSlot(leftPg, ln+i, k, p, l)
		}
		setNumKeys(leftPg, ln+rn)
		newNext := binary.LittleEndian.Uint64(rightPg[offNextLeaf : offNextLeaf+8])
		binary.LittleEndian.PutUint64(leftPg[offNextLeaf:], newNext)
		if newNext != pager.InvalidPage {
			nextPg, err := t.pg.Read(newNext)
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint64(nextPg[offPrevLeaf:], leftID)
			if err := t.pg.Write(newNext, nextPg); err != nil {
				return err
			}
		}
		if err := t.pg.Write(leftID, leftPg); err != nil {
			return err
		}
	} else {
		// Internal: pull separator down.
		sepKey, _, _ := getSlot(parentPg, sepIdx)
		rightFirstChild := binary.LittleEndian.Uint64(rightPg[offFirstPtr : offFirstPtr+8])
		putSlot(leftPg, ln, sepKey, rightFirstChild, 0)
		ln++
		for i := 0; i < rn; i++ {
			k, p, l := getSlot(rightPg, i)
			putSlot(leftPg, ln+i, k, p, l)
		}
		setNumKeys(leftPg, ln+rn)
		if err := t.pg.Write(leftID, leftPg); err != nil {
			return err
		}
	}

	// Remove separator from parent.
	for i := sepIdx; i < parentN-1; i++ {
		k, p, l := getSlot(parentPg, i+1)
		putSlot(parentPg, i, k, p, l)
	}
	setNumKeys(parentPg, parentN-1)
	return t.pg.Write(parentID, parentPg)
}

// ─── Value heap ───────────────────────────────────────────────────────────────

func (t *BPTree) appendValue(value []byte) (int64, error) {
	offset := t.valSize
	_, err := t.valFile.WriteAt(value, offset)
	if err != nil {
		return 0, fmt.Errorf("bptree: append value: %w", err)
	}
	t.valSize += int64(len(value))
	return offset, nil
}

func (t *BPTree) readValue(offset int64, length uint32) ([]byte, error) {
	buf := make([]byte, length)
	_, err := t.valFile.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("bptree: read value: %w", err)
	}
	return buf, nil
}

// ─── Header ───────────────────────────────────────────────────────────────────

func (t *BPTree) writeHeader() error {
	pg, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(pg[:8], t.rootID)
	return t.pg.Write(1, pg)
}

func (t *BPTree) readHeader() error {
	pg, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	t.rootID = binary.LittleEndian.Uint64(pg[:8])
	return nil
}

// ─── Range Iterator ───────────────────────────────────────────────────────────

// RangeIterator scans leaves using the linked-list pointer.
// This is the core advantage of B+ trees over B-trees for range scans.
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

// Next advances the iterator. Returns false when exhausted.
func (it *RangeIterator) Next() bool {
	if it.done || it.leafID == pager.InvalidPage {
		return false
	}
	for {
		pg, err := it.tree.pg.Read(it.leafID)
		if err != nil {
			it.err = err
			return false
		}
		n := numKeys(pg)

		if it.idx < n {
			k, valOffset, valLen := getSlot(pg, it.idx)
			if k > it.end {
				it.done = true
				return false
			}
			it.idx++
			val, err := it.tree.readValue(int64(valOffset), valLen)
			if err != nil {
				it.err = err
				return false
			}
			it.key = k
			it.val = val
			return true
		}

		// Move to next leaf via the linked list pointer — O(1), no tree traversal.
		nextID := binary.LittleEndian.Uint64(pg[offNextLeaf : offNextLeaf+8])
		if nextID == pager.InvalidPage {
			it.done = true
			return false
		}
		it.leafID = nextID
		it.idx = 0
	}
}

func (it *RangeIterator) Key() int64    { return it.key }
func (it *RangeIterator) Value() []byte { return it.val }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
