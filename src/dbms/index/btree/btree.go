// Package btree implements a disk-based B-tree using the pager package.
//
// Page layout (4096 bytes):
//
//	[0]      uint8   — node type (0 = internal, 1 = leaf)
//	[1..2]   uint16  — number of keys currently in this node
//	[3..10]  uint64  — page ID of the first child (internal only; unused in leaf)
//
// Then for each key slot i in [0, order-1):
//
//	keys[i]      int64  (8 bytes)
//	children[i+1] uint64 (8 bytes)  — right child of keys[i] (internal only)
//	values[i]    — for leaf: 8-byte offset + 4-byte length into a separate value file
//
// To keep the page layout simple, values are stored in a separate append-only file.
package btree

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/pager"
)

// ─── Constants ────────────────────────────────────────────────────────────────

// order is the maximum number of children an internal node can have.
// Derived from page size so that a node fits exactly in one page.
//
// Internal node layout per slot: 8 (key) + 8 (child ptr) = 16 bytes
// Header: 1 (type) + 2 (numKeys) + 8 (first child) = 11 bytes
// Remaining: 4096 - 11 = 4085 bytes → 4085 / 16 = 255 slots → order = 256
//
// Leaf node layout per slot: 8 (key) + 8 (value offset) + 4 (value len) = 20 bytes
// Header: 1 + 2 = 3 bytes → (4096 - 3) / 20 = 204 slots
//
// We use the minimum of the two so that the same order constant works for both.
const (
	order   = 200 // max children per internal node (keys = order-1)
	maxKeys = order - 1
	minKeys = (order - 1) / 2

	typeInternal = byte(0)
	typeLeaf     = byte(1)

	// Offsets inside a raw page.
	offType     = 0
	offNumKeys  = 1  // uint16, 2 bytes
	offFirstPtr = 3  // uint64, 8 bytes (internal: first child; leaf: unused)
	offSlots    = 11 // start of key/ptr/value slots
)

// slotSize is different for internal vs leaf nodes, but we keep a unified
// layout to simplify reading: every slot is 20 bytes regardless.
// Internal nodes ignore the last 4 bytes of each slot (value length field).
//
// Slot layout: [key int64 8B][ptr/offset uint64 8B][valueLen uint32 4B]
const slotSize = 20

// ─── BTree ────────────────────────────────────────────────────────────────────

// BTree is a disk-based B-tree.
type BTree struct {
	pg      *pager.Pager
	valFile *os.File // append-only value heap
	rootID  uint64
	valSize int64 // current end of value file
}

// Open opens (or creates) a B-tree stored at the given base path.
// Two files are created: <path>.bt (pages) and <path>.bv (values).
// cachePages controls the LRU page cache size (e.g. 256 = 1MB, 4096 = 16MB).
func Open(path string, cachePages int) (*BTree, error) {
	pg, err := pager.Open(path+".bt", cachePages)
	if err != nil {
		return nil, err
	}

	vf, err := os.OpenFile(path+".bv", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		pg.Close()
		return nil, err
	}

	info, err := vf.Stat()
	if err != nil {
		return nil, err
	}

	t := &BTree{pg: pg, valFile: vf, valSize: info.Size()}

	// Page 0 is the pager header. Page 1 is the B-tree's own header page,
	// which stores the root page ID.
	if pg.PageCount() <= 2 {
		// Brand new tree — allocate header page and root leaf.
		_, err = pg.Allocate() // page 1: btree header
		if err != nil {
			return nil, err
		}
		rootID, err := pg.Allocate() // page 2: initial root (leaf)
		if err != nil {
			return nil, err
		}
		t.rootID = rootID
		if err := t.initLeaf(rootID); err != nil {
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

// Close flushes and closes all underlying files.
func (t *BTree) Close() error {
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
func (t *BTree) Insert(key int64, value []byte) error {
	// Append value to heap file.
	offset, err := t.appendValue(value)
	if err != nil {
		return err
	}

	// Insert into the tree. If the root splits, create a new root.
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

// Get retrieves the value associated with key.
func (t *BTree) Get(key int64) ([]byte, error) {
	return t.searchNode(t.rootID, key)
}

// Delete removes the key from the tree. Returns nil if key not found.
func (t *BTree) Delete(key int64) error {
	found, err := t.deleteNode(t.rootID, key)
	if err != nil {
		return err
	}
	if !found {
		return nil // key not present — not an error for benchmarking
	}
	// If root is now empty internal node, collapse it.
	pg, err := t.pg.Read(t.rootID)
	if err != nil {
		return err
	}
	nk := int(binary.LittleEndian.Uint16(pg[offNumKeys : offNumKeys+2]))
	if nk == 0 && pg[offType] == typeInternal {
		// New root is the first child.
		t.rootID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		if err := t.writeHeader(); err != nil {
			return err
		}
	}
	return nil
}

// Range returns an iterator over all keys in [start, end].
func (t *BTree) Range(start, end int64) (index.Iterator, error) {
	it := &RangeIterator{tree: t, end: end}
	if err := it.seekToFirst(t.rootID, start); err != nil {
		return nil, err
	}
	return it, nil
}

// ─── Node helpers ─────────────────────────────────────────────────────────────

func (t *BTree) initLeaf(id uint64) error {
	pg := new(pager.Page)
	pg[offType] = typeLeaf
	binary.LittleEndian.PutUint16(pg[offNumKeys:], 0)
	return t.pg.Write(id, pg)
}

func (t *BTree) initInternal(id, leftChild uint64, key int64, rightChild uint64) error {
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

// findKeyIndex returns the index of the first key >= key (binary search).
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

// ─── Insert logic ─────────────────────────────────────────────────────────────

// insertNode inserts (key, valOffset, valLen) into the subtree rooted at nodeID.
// Returns (midKey, newRightPageID, didSplit, error).
func (t *BTree) insertNode(nodeID uint64, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
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

func (t *BTree) insertLeaf(nodeID uint64, pg *pager.Page, n int, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
	idx := findKeyIndex(pg, key, n)

	// Update existing key.
	if idx < n {
		if k, _, _ := getSlot(pg, idx); k == key {
			putSlot(pg, idx, key, uint64(valOffset), valLen)
			return 0, 0, false, t.pg.Write(nodeID, pg)
		}
	}

	// Shift right to make room.
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

func (t *BTree) splitLeaf(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, err := t.pg.Allocate()
	if err != nil {
		return 0, 0, false, err
	}
	newPg := new(pager.Page)
	newPg[offType] = typeLeaf

	// Move right half to new page.
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

	midKey, _, _ := getSlot(newPg, 0)
	return midKey, newID, true, nil
}

func (t *BTree) insertInternal(nodeID uint64, pg *pager.Page, n int, key int64, valOffset int64, valLen uint32) (int64, uint64, bool, error) {
	idx := findKeyIndex(pg, key, n)

	// Choose child.
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

	// Insert midKey + newChildID into this internal node.
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

func (t *BTree) splitInternal(nodeID uint64, pg *pager.Page, n int) (int64, uint64, bool, error) {
	mid := n / 2
	newID, err := t.pg.Allocate()
	if err != nil {
		return 0, 0, false, err
	}
	newPg := new(pager.Page)
	newPg[offType] = typeInternal

	midKey, midRightChild, _ := getSlot(pg, mid)

	// New node's first child = midRightChild.
	binary.LittleEndian.PutUint64(newPg[offFirstPtr:], midRightChild)

	// Copy right half (after mid) to new page.
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

// ─── Search ───────────────────────────────────────────────────────────────────

func (t *BTree) searchNode(nodeID uint64, key int64) ([]byte, error) {
	for {
		pg, err := t.pg.Read(nodeID)
		if err != nil {
			return nil, err
		}
		n := numKeys(pg)

		if pg[offType] == typeLeaf {
			idx := findKeyIndex(pg, key, n)
			if idx < n {
				if k, valOffset, valLen := getSlot(pg, idx); k == key {
					return t.readValue(int64(valOffset), valLen)
				}
			}
			return nil, nil
		}

		// Internal node: find child.
		idx := findKeyIndex(pg, key, n)
		if idx == 0 {
			nodeID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		} else {
			_, nodeID, _ = getSlot(pg, idx-1)
		}
	}
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// deleteNode recursively deletes key from the subtree.
// Returns (found, error).
func (t *BTree) deleteNode(nodeID uint64, key int64) (bool, error) {
	pg, err := t.pg.Read(nodeID)
	if err != nil {
		return false, err
	}
	n := numKeys(pg)

	if pg[offType] == typeLeaf {
		return t.deleteFromLeaf(nodeID, pg, n, key)
	}

	// childPos = which child slot to descend into.
	// idx==0 means firstPtr (childPos=0); otherwise right child of slot idx-1 (childPos=idx).
	childPos := findKeyIndex(pg, key, n)
	var childID uint64
	if childPos == 0 {
		childID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	} else {
		_, childID, _ = getSlot(pg, childPos-1)
	}

	found, err := t.deleteNode(childID, key)
	if err != nil || !found {
		return found, err
	}

	// Re-read child — it may have changed on disk.
	childPg, err := t.pg.Read(childID)
	if err != nil {
		return true, err
	}
	if numKeys(childPg) >= minKeys {
		return true, nil
	}

	// Child is underflowing — re-read parent (recursive writes may have changed it).
	parentPg, err := t.pg.Read(nodeID)
	if err != nil {
		return true, err
	}
	parentN := numKeys(parentPg)
	return true, t.rebalance(nodeID, parentPg, parentN, childPos, childID)
}

func (t *BTree) deleteFromLeaf(nodeID uint64, pg *pager.Page, n int, key int64) (bool, error) {
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

// childPageID returns the page ID of child at position pos in an internal node.
// pos=0 → firstPtr, pos=i → right-child ptr of slot i-1.
func childPageID(pg *pager.Page, pos int) uint64 {
	if pos == 0 {
		return binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
	}
	_, id, _ := getSlot(pg, pos-1)
	return id
}

// rebalance fixes an underflowing child at childPos by either borrowing a key
// from a sibling (if the sibling has surplus keys) or merging with a sibling.
func (t *BTree) rebalance(parentID uint64, parentPg *pager.Page, parentN int, childPos int, childID uint64) error {
	// Try right sibling first.
	if childPos < parentN {
		rightID := childPageID(parentPg, childPos+1)
		rightPg, err := t.pg.Read(rightID)
		if err != nil {
			return err
		}
		if numKeys(rightPg) > minKeys {
			// Borrow from right sibling.
			return t.borrowFromRight(parentID, parentPg, childPos, childID, rightID, rightPg)
		}
		// Merge child (left) with right sibling.
		return t.mergeNodes(parentID, parentPg, parentN, childPos, childID, rightID)
	}

	// No right sibling — try left.
	leftID := childPageID(parentPg, childPos-1)
	leftPg, err := t.pg.Read(leftID)
	if err != nil {
		return err
	}
	if numKeys(leftPg) > minKeys {
		// Borrow from left sibling.
		return t.borrowFromLeft(parentID, parentPg, childPos-1, leftID, leftPg, childID)
	}
	// Merge left sibling with child (right).
	return t.mergeNodes(parentID, parentPg, parentN, childPos-1, leftID, childID)
}

// borrowFromRight rotates one key from the right sibling through the parent into the child.
func (t *BTree) borrowFromRight(parentID uint64, parentPg *pager.Page, sepIdx int, childID, rightID uint64, rightPg *pager.Page) error {
	childPg, err := t.pg.Read(childID)
	if err != nil {
		return err
	}
	cn := numKeys(childPg)
	rn := numKeys(rightPg)

	sepKey, _, _ := getSlot(parentPg, sepIdx)

	if childPg[offType] == typeInternal {
		// Append separator to child; right's firstPtr becomes child's new last right-child.
		rightFirstChild := binary.LittleEndian.Uint64(rightPg[offFirstPtr : offFirstPtr+8])
		putSlot(childPg, cn, sepKey, rightFirstChild, 0)
		setNumKeys(childPg, cn+1)
		// New separator = right's first key; right's new firstPtr = right's first right-child.
		newSepKey, newFirstChild, _ := getSlot(rightPg, 0)
		binary.LittleEndian.PutUint64(rightPg[offFirstPtr:], newFirstChild)
		for i := 0; i < rn-1; i++ {
			k, p, l := getSlot(rightPg, i+1)
			putSlot(rightPg, i, k, p, l)
		}
		setNumKeys(rightPg, rn-1)
		// Update separator in parent.
		putSlot(parentPg, sepIdx, newSepKey, rightID, 0)
	} else {
		// Leaf: move right's first key into child.
		newKey, newPtr, newLen := getSlot(rightPg, 0)
		putSlot(childPg, cn, newKey, newPtr, newLen)
		setNumKeys(childPg, cn+1)
		for i := 0; i < rn-1; i++ {
			k, p, l := getSlot(rightPg, i+1)
			putSlot(rightPg, i, k, p, l)
		}
		setNumKeys(rightPg, rn-1)
		// Update separator in parent to right's new first key.
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

// borrowFromLeft rotates one key from the left sibling through the parent into the child.
func (t *BTree) borrowFromLeft(parentID uint64, parentPg *pager.Page, sepIdx int, leftID uint64, leftPg *pager.Page, childID uint64) error {
	childPg, err := t.pg.Read(childID)
	if err != nil {
		return err
	}
	cn := numKeys(childPg)
	ln := numKeys(leftPg)

	sepKey, _, _ := getSlot(parentPg, sepIdx)

	// Shift child's keys right by one to make room at index 0.
	if childPg[offType] == typeInternal {
		oldFirstChild := binary.LittleEndian.Uint64(childPg[offFirstPtr : offFirstPtr+8])
		for i := cn; i > 0; i-- {
			k, p, l := getSlot(childPg, i-1)
			putSlot(childPg, i, k, p, l)
		}
		// slot 0 gets separator; firstPtr gets left's last right-child.
		lastKey, lastChild, _ := getSlot(leftPg, ln-1)
		putSlot(childPg, 0, sepKey, oldFirstChild, 0)
		binary.LittleEndian.PutUint64(childPg[offFirstPtr:], lastChild)
		setNumKeys(childPg, cn+1)
		setNumKeys(leftPg, ln-1)
		// New separator = left's last key.
		putSlot(parentPg, sepIdx, lastKey, childID, 0)
	} else {
		// Leaf: shift right, prepend left's last entry.
		for i := cn; i > 0; i-- {
			k, p, l := getSlot(childPg, i-1)
			putSlot(childPg, i, k, p, l)
		}
		lastKey, lastPtr, lastLen := getSlot(leftPg, ln-1)
		putSlot(childPg, 0, lastKey, lastPtr, lastLen)
		setNumKeys(childPg, cn+1)
		setNumKeys(leftPg, ln-1)
		// New separator = child's new first key.
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

// mergeNodes merges right into left, pulling the separator down from the parent.
// Only called when left.n + right.n + 1 (separator) <= maxKeys.
func (t *BTree) mergeNodes(parentID uint64, parentPg *pager.Page, parentN int, sepIdx int, leftID, rightID uint64) error {
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

	if leftPg[offType] == typeInternal {
		sepKey, _, _ := getSlot(parentPg, sepIdx)
		rightFirstChild := binary.LittleEndian.Uint64(rightPg[offFirstPtr : offFirstPtr+8])
		putSlot(leftPg, ln, sepKey, rightFirstChild, 0)
		ln++
		for i := 0; i < rn; i++ {
			k, p, l := getSlot(rightPg, i)
			putSlot(leftPg, ln+i, k, p, l)
		}
		setNumKeys(leftPg, ln+rn)
	} else {
		for i := 0; i < rn; i++ {
			k, p, l := getSlot(rightPg, i)
			putSlot(leftPg, ln+i, k, p, l)
		}
		setNumKeys(leftPg, ln+rn)
	}

	if err := t.pg.Write(leftID, leftPg); err != nil {
		return err
	}

	// Remove separator from parent; left now covers the range of both.
	for i := sepIdx; i < parentN-1; i++ {
		k, p, l := getSlot(parentPg, i+1)
		putSlot(parentPg, i, k, p, l)
	}
	setNumKeys(parentPg, parentN-1)
	return t.pg.Write(parentID, parentPg)
}

// ─── Value heap ───────────────────────────────────────────────────────────────

func (t *BTree) appendValue(value []byte) (int64, error) {
	offset := t.valSize
	_, err := t.valFile.WriteAt(value, offset)
	if err != nil {
		return 0, fmt.Errorf("btree: append value: %w", err)
	}
	t.valSize += int64(len(value))
	return offset, nil
}

func (t *BTree) readValue(offset int64, length uint32) ([]byte, error) {
	buf := make([]byte, length)
	_, err := t.valFile.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("btree: read value: %w", err)
	}
	return buf, nil
}

// ─── Header ───────────────────────────────────────────────────────────────────

func (t *BTree) writeHeader() error {
	pg, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(pg[:8], t.rootID)
	return t.pg.Write(1, pg)
}

func (t *BTree) readHeader() error {
	pg, err := t.pg.Read(1)
	if err != nil {
		return err
	}
	t.rootID = binary.LittleEndian.Uint64(pg[:8])
	return nil
}

// ─── Range Iterator ───────────────────────────────────────────────────────────

// RangeIterator scans a B-tree from a start key to an end key inclusive.
// Because B-trees don't have linked leaves, we use a stack-based in-order traversal.
type RangeIterator struct {
	tree  *BTree
	end   int64
	stack []stackFrame
	key   int64
	val   []byte
	err   error
	done  bool
}

type stackFrame struct {
	pageID uint64
	idx    int // next child index to visit (0 = visit firstPtr child next)
}

func (it *RangeIterator) seekToFirst(rootID uint64, start int64) error {
	// Walk down to the leaf containing start, pushing frames onto the stack.
	nodeID := rootID
	for {
		pg, err := it.tree.pg.Read(nodeID)
		if err != nil {
			return err
		}
		n := numKeys(pg)
		idx := findKeyIndex(pg, start, n)

		if pg[offType] == typeLeaf {
			it.stack = append(it.stack, stackFrame{pageID: nodeID, idx: idx})
			return nil
		}

		// Push this internal node so we can resume scanning it later.
		it.stack = append(it.stack, stackFrame{pageID: nodeID, idx: idx})

		// Descend into appropriate child.
		if idx == 0 {
			nodeID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		} else {
			_, nodeID, _ = getSlot(pg, idx-1)
		}
	}
}

// Next advances the iterator. Returns false when exhausted.
func (it *RangeIterator) Next() bool {
	if it.done || len(it.stack) == 0 {
		return false
	}
	for len(it.stack) > 0 {
		frame := &it.stack[len(it.stack)-1]
		pg, err := it.tree.pg.Read(frame.pageID)
		if err != nil {
			it.err = err
			return false
		}
		n := numKeys(pg)

		if pg[offType] == typeLeaf {
			if frame.idx >= n {
				it.stack = it.stack[:len(it.stack)-1]
				continue
			}
			k, valOffset, valLen := getSlot(pg, frame.idx)
			if k > it.end {
				it.done = true
				return false
			}
			frame.idx++
			val, err := it.tree.readValue(int64(valOffset), valLen)
			if err != nil {
				it.err = err
				return false
			}
			it.key = k
			it.val = val
			return true
		}

		// Internal node: interleave children and keys.
		// frame.idx tracks the next key to emit.
		// Before emitting key[i] we must have visited child[i].
		// After emitting key[n-1] we visit child[n] (rightmost).
		//
		// Strategy: on each visit push the next child, emit the separator key.
		if frame.idx > n {
			it.stack = it.stack[:len(it.stack)-1]
			continue
		}

		// Determine next child pageID.
		var childID uint64
		if frame.idx == 0 {
			childID = binary.LittleEndian.Uint64(pg[offFirstPtr : offFirstPtr+8])
		} else {
			_, childID, _ = getSlot(pg, frame.idx-1)
		}
		frame.idx++

		// Push child subtree.
		it.stack = append(it.stack, stackFrame{pageID: childID, idx: 0})
	}
	return false
}

func (it *RangeIterator) Key() int64    { return it.key }
func (it *RangeIterator) Value() []byte { return it.val }
func (it *RangeIterator) Error() error  { return it.err }
func (it *RangeIterator) Close() error  { return nil }
