// Package btpage defines the layout and management of on-disk pages for B-tree and B+ tree implementations.
//
// The page layout is designed for efficient storage and access:
// [0]     1 byte   page type (TypeInternal / TypeLeaf)
// [1-2]   2 bytes  numCells (number of items on the page)
// [3-4]   2 bytes  cellContentStart (offset to the top of the cell area)
// [5-8]   4 bytes  rightmost child page ID (internal pages only)
// [9-12]  4 bytes  nextLeaf page ID (B+ tree leaf linkage)
// [13+]   cell pointer array (uint16 offsets growing downward)
package btpage

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/pager"
)

const (
	// TypeInternal represents an internal node in the tree.
	TypeInternal = byte(0)
	// TypeLeaf represents a leaf node in the tree.
	TypeLeaf = byte(1)

	// Offsets for various fields in the page header.
	OffType        = 0
	OffNumCells    = 1
	OffCellContent = 3
	OffRightmost   = 5
	OffNextLeaf    = 9
	OffCellPtrs    = 13

	// CellPtrSize is the size of each cell pointer in bytes.
	CellPtrSize = 2

	// InvalidPage is the constant used for representing an invalid page ID.
	InvalidPage = uint32(0xFFFFFFFF)
)

// InitPage initializes a new page with the given type.
func InitPage(p pager.Page, pt byte) {
	for i := range p {
		p[i] = 0
	}
	p[OffType] = pt
	SetNumCells(p, 0)
	SetCellContent(p, uint16(len(p)))
	SetNextLeaf(p, InvalidPage)
}

// NumCells returns the number of cells stored on the page.
func NumCells(p pager.Page) int {
	return int(binary.LittleEndian.Uint16(p[OffNumCells : OffNumCells+2]))
}

// SetNumCells sets the number of cells stored on the page.
func SetNumCells(p pager.Page, n int) {
	binary.LittleEndian.PutUint16(p[OffNumCells:OffNumCells+2], uint16(n))
}

// CellContent returns the offset to the beginning of the cell content area.
func CellContent(p pager.Page) uint16 {
	return binary.LittleEndian.Uint16(p[OffCellContent : OffCellContent+2])
}

// SetCellContent sets the offset to the beginning of the cell content area.
func SetCellContent(p pager.Page, v uint16) {
	binary.LittleEndian.PutUint16(p[OffCellContent:OffCellContent+2], v)
}

// Rightmost returns the page ID of the rightmost child.
func Rightmost(p pager.Page) uint32 {
	return binary.LittleEndian.Uint32(p[OffRightmost : OffRightmost+4])
}

// SetRightmost sets the page ID of the rightmost child.
func SetRightmost(p pager.Page, id uint32) {
	binary.LittleEndian.PutUint32(p[OffRightmost:OffRightmost+4], id)
}

// NextLeaf returns the page ID of the next leaf page.
func NextLeaf(p pager.Page) uint32 {
	return binary.LittleEndian.Uint32(p[OffNextLeaf : OffNextLeaf+4])
}

// SetNextLeaf sets the page ID of the next leaf page.
func SetNextLeaf(p pager.Page, id uint32) {
	binary.LittleEndian.PutUint32(p[OffNextLeaf:OffNextLeaf+4], id)
}

// CellPtr returns the offset to the i-th cell.
func CellPtr(p pager.Page, i int) uint16 {
	o := OffCellPtrs + i*CellPtrSize
	return binary.LittleEndian.Uint16(p[o : o+2])
}

// SetCellPtr sets the offset for the i-th cell.
func SetCellPtr(p pager.Page, i int, off uint16) {
	o := OffCellPtrs + i*CellPtrSize
	binary.LittleEndian.PutUint16(p[o:o+2], off)
}

// FreeSpace calculates the remaining free space on the page.
func FreeSpace(p pager.Page, n int) int {
	return int(CellContent(p)) - (OffCellPtrs + n*CellPtrSize)
}

// AllocCell allocates space for a cell of the given size and returns its offset.
func AllocCell(p pager.Page, size int) int {
	top := int(CellContent(p)) - size
	SetCellContent(p, uint16(top))
	return top
}
