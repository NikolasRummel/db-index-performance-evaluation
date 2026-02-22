package pager

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	PageSize    = 4096 // 4 KB — matches OS page size
	InvalidPage = ^uint64(0)
)

// Page is a raw 4 KB block read from or written to disk.
type Page [PageSize]byte

// Pager manages a file of fixed-size pages and caches recently used ones.
type Pager struct {
	file      *os.File
	cache     *lruCache
	pageCount uint64 // total number of pages ever allocated
}

// Open opens (or creates) a pager backed by the given file.
// cacheSize is the number of pages to hold in the LRU cache.
func Open(path string, cacheSize int) (*Pager, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("pager open: %w", err)
	}

	p := &Pager{
		file:  f,
		cache: newLRUCache(cacheSize),
	}

	// Read the page count from the file header (first 8 bytes of page 0).
	// If the file is brand new, pageCount starts at 1 (page 0 is the header).
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		p.pageCount = 1
		if err := p.writePageCount(); err != nil {
			return nil, err
		}
	} else {
		pg, err := p.readPageFromDisk(0)
		if err != nil {
			return nil, fmt.Errorf("pager: read header: %w", err)
		}
		p.pageCount = binary.LittleEndian.Uint64(pg[:8])
	}

	return p, nil
}

// Allocate reserves a new page on disk and returns its page ID.
func (p *Pager) Allocate() (uint64, error) {
	id := p.pageCount
	p.pageCount++

	// Write an empty page to extend the file.
	var blank Page
	if err := p.writePageToDisk(id, &blank); err != nil {
		return 0, err
	}
	if err := p.writePageCount(); err != nil {
		return 0, err
	}
	return id, nil
}

// Read returns the page with the given ID, from cache or disk.
func (p *Pager) Read(id uint64) (*Page, error) {
	if pg := p.cache.get(id); pg != nil {
		return pg, nil
	}
	pg, err := p.readPageFromDisk(id)
	if err != nil {
		return nil, err
	}
	p.cache.put(id, pg)
	return pg, nil
}

// Write writes a page back to disk and updates the cache.
func (p *Pager) Write(id uint64, pg *Page) error {
	p.cache.put(id, pg)
	return p.writePageToDisk(id, pg)
}

// Close flushes and closes the underlying file.
func (p *Pager) Close() error {
	return p.file.Close()
}

// PageCount returns the total number of allocated pages.
func (p *Pager) PageCount() uint64 {
	return p.pageCount
}

// --- internal helpers ---

func (p *Pager) offset(id uint64) int64 {
	return int64(id) * PageSize
}

func (p *Pager) readPageFromDisk(id uint64) (*Page, error) {
	pg := new(Page)
	_, err := p.file.ReadAt(pg[:], p.offset(id))
	if err != nil {
		return nil, fmt.Errorf("pager: read page %d: %w", id, err)
	}
	return pg, nil
}

func (p *Pager) writePageToDisk(id uint64, pg *Page) error {
	_, err := p.file.WriteAt(pg[:], p.offset(id))
	if err != nil {
		return fmt.Errorf("pager: write page %d: %w", id, err)
	}
	return nil
}

func (p *Pager) writePageCount() error {
	var hdr Page
	// Preserve existing header content if the file already has data.
	if p.pageCount > 1 {
		existing, err := p.readPageFromDisk(0)
		if err == nil {
			hdr = *existing
		}
	}
	binary.LittleEndian.PutUint64(hdr[:8], p.pageCount)
	return p.writePageToDisk(0, &hdr)
}

// ─── LRU Cache ────────────────────────────────────────────────────────────────

type lruEntry struct {
	id   uint64
	page *Page
	prev *lruEntry
	next *lruEntry
}

type lruCache struct {
	cap   int
	items map[uint64]*lruEntry
	head  *lruEntry // most recent
	tail  *lruEntry // least recent
}

func newLRUCache(cap int) *lruCache {
	return &lruCache{
		cap:   cap,
		items: make(map[uint64]*lruEntry, cap),
	}
}

func (c *lruCache) get(id uint64) *Page {
	e, ok := c.items[id]
	if !ok {
		return nil
	}
	c.moveToFront(e)
	return e.page
}

func (c *lruCache) put(id uint64, pg *Page) {
	if e, ok := c.items[id]; ok {
		e.page = pg
		c.moveToFront(e)
		return
	}
	e := &lruEntry{id: id, page: pg}
	c.items[id] = e
	c.pushFront(e)
	if len(c.items) > c.cap {
		c.evict()
	}
}

func (c *lruCache) pushFront(e *lruEntry) {
	e.next = c.head
	e.prev = nil
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *lruCache) moveToFront(e *lruEntry) {
	if c.head == e {
		return
	}
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if c.tail == e {
		c.tail = e.prev
	}
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
}

func (c *lruCache) evict() {
	if c.tail == nil {
		return
	}
	delete(c.items, c.tail.id)
	if c.tail.prev != nil {
		c.tail.prev.next = nil
	}
	c.tail = c.tail.prev
	if c.tail == nil {
		c.head = nil
	}
}
