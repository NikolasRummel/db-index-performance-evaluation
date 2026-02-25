// Package btpage provides the shared on-disk page layout used by both the
// B-tree and B+ tree implementations.
//
// Page layout:
//
//	[0]     1 byte   page type (TypeInternal / TypeLeaf)
//	[1-2]   2 bytes  numCells
//	[3-4]   2 bytes  cellContentStart (top of cell area, grows upward from bottom)
//	[5-8]   4 bytes  rightmost child page ID (internal pages only)
//	[9-12]  4 bytes  nextLeaf page ID (B+ tree leaf pages only, else InvalidPage)
//	[13+]   cell pointer array — one uint16 offset per cell, grows downward
//	        ...free space...
//	        cell content area, grows upward from bottom of page
package btpage

import (
	"encoding/binary"

	"github.com/btree-query-bench/bmark/dbms/pager"
)

const (
	TypeInternal = byte(0)
	TypeLeaf     = byte(1)

	OffType        = 0
	OffNumCells    = 1
	OffCellContent = 3
	OffRightmost   = 5
	OffNextLeaf    = 9
	OffCellPtrs    = 13

	CellPtrSize = 2

	InvalidPage = uint32(0xFFFFFFFF)
)

func InitPage(p *pager.Page, pt byte) {
	for i := range p {
		p[i] = 0
	}
	p[OffType] = pt
	SetNumCells(p, 0)
	SetCellContent(p, uint16(pager.PageSize))
	SetNextLeaf(p, InvalidPage)
}

func NumCells(p *pager.Page) int {
	return int(binary.LittleEndian.Uint16(p[OffNumCells : OffNumCells+2]))
}

func SetNumCells(p *pager.Page, n int) {
	binary.LittleEndian.PutUint16(p[OffNumCells:OffNumCells+2], uint16(n))
}

func CellContent(p *pager.Page) uint16 {
	return binary.LittleEndian.Uint16(p[OffCellContent : OffCellContent+2])
}

func SetCellContent(p *pager.Page, v uint16) {
	binary.LittleEndian.PutUint16(p[OffCellContent:OffCellContent+2], v)
}

func Rightmost(p *pager.Page) uint32 {
	return binary.LittleEndian.Uint32(p[OffRightmost : OffRightmost+4])
}

func SetRightmost(p *pager.Page, id uint32) {
	binary.LittleEndian.PutUint32(p[OffRightmost:OffRightmost+4], id)
}

func NextLeaf(p *pager.Page) uint32 {
	return binary.LittleEndian.Uint32(p[OffNextLeaf : OffNextLeaf+4])
}

func SetNextLeaf(p *pager.Page, id uint32) {
	binary.LittleEndian.PutUint32(p[OffNextLeaf:OffNextLeaf+4], id)
}

func CellPtr(p *pager.Page, i int) uint16 {
	o := OffCellPtrs + i*CellPtrSize
	return binary.LittleEndian.Uint16(p[o : o+2])
}

func SetCellPtr(p *pager.Page, i int, off uint16) {
	o := OffCellPtrs + i*CellPtrSize
	binary.LittleEndian.PutUint16(p[o:o+2], off)
}

func FreeSpace(p *pager.Page, n int) int {
	return int(CellContent(p)) - (OffCellPtrs + n*CellPtrSize)
}

func AllocCell(p *pager.Page, size int) int {
	top := int(CellContent(p)) - size
	SetCellContent(p, uint16(top))
	return top
}
