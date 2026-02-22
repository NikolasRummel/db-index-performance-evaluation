// Package lsm wraps Pebble (CockroachDB's LSM storage engine) behind the
// common Index interface so it can be benchmarked alongside the custom
// B-tree and B+ tree implementations.
package lsm

import (
	"encoding/binary"
	"fmt"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/cockroachdb/pebble"
)

type LSM struct {
	db *pebble.DB
}

// Open opens (or creates) a Pebble database at the given directory path.
func Open(dir string) (*LSM, error) {
	opts := &pebble.Options{
		// Use a 64 MB memtable
		MemTableSize: 16 << 20,
		// Keep 2 memtables so one can be flushed while the other is active.
		MemTableStopWritesThreshold: 4,
		// L0 compaction trigger.
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
	}

	db, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, fmt.Errorf("lsm: open: %w", err)
	}
	return &LSM{db: db}, nil
}

// Close cleanly shuts down Pebble, flushing any in-memory state.
func (l *LSM) Close() error {
	return l.db.Close()
}

// Insert inserts or updates the value for key.
func (l *LSM) Insert(key int64, value []byte) error {
	return l.db.Set(encodeKey(key), value, pebble.NoSync)
}

// Get retrieves the value for key. Returns nil if not found.
func (l *LSM) Get(key int64) ([]byte, error) {
	val, closer, err := l.db.Get(encodeKey(key))
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lsm: get: %w", err)
	}
	// val is only valid until closer.Close(), so we copy it.
	result := make([]byte, len(val))
	copy(result, val)
	closer.Close()
	return result, nil
}

// Delete removes the key from the store.
func (l *LSM) Delete(key int64) error {
	err := l.db.Delete(encodeKey(key), pebble.NoSync)
	if err != nil {
		return fmt.Errorf("lsm: delete: %w", err)
	}
	return nil
}

// Range returns an iterator over all keys in [start, end] inclusive.
func (l *LSM) Range(start, end int64) (index.Iterator, error) {
	iterOpts := &pebble.IterOptions{
		LowerBound: encodeKey(start),
		UpperBound: encodeKeyExclusive(end),
	}
	iter, err := l.db.NewIter(iterOpts)
	if err != nil {
		return nil, fmt.Errorf("lsm: range: %w", err)
	}
	iter.First()
	return &rangeIterator{iter: iter, first: true}, nil
}

// ─── Key encoding ─────────────────────────────────────────────────────────────

// encodeKey encodes an int64 as a big-endian 8-byte slice.
// Big-endian preserves sort order, which Pebble (and all LSM trees) rely on.
func encodeKey(k int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(k))
	return b
}

// encodeKeyExclusive returns the exclusive upper bound for use with Pebble's
// UpperBound option (which is exclusive, unlike our interface which is inclusive).
func encodeKeyExclusive(k int64) []byte {
	return encodeKey(k + 1)
}

// ─── Range Iterator ───────────────────────────────────────────────────────────

type rangeIterator struct {
	iter  *pebble.Iterator
	first bool
	key   int64
	val   []byte
	err   error
}

func (it *rangeIterator) Next() bool {
	var valid bool
	if it.first {
		// iter.First() was already called in Range(); just check validity.
		it.first = false
		valid = it.iter.Valid()
	} else {
		valid = it.iter.Next()
	}
	if !valid {
		return false
	}
	k := it.iter.Key()
	if len(k) != 8 {
		it.err = fmt.Errorf("lsm: unexpected key length %d", len(k))
		return false
	}
	it.key = int64(binary.BigEndian.Uint64(k))
	// Copy value — Pebble reuses the buffer on Next().
	v := it.iter.Value()
	it.val = make([]byte, len(v))
	copy(it.val, v)
	return true
}

func (it *rangeIterator) Key() int64    { return it.key }
func (it *rangeIterator) Value() []byte { return it.val }
func (it *rangeIterator) Error() error  { return it.err }
func (it *rangeIterator) Close() error  { return it.iter.Close() }
