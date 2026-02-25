package shared

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"

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
		// Leaf nodes have no child pointers.
		leftChild := uint32(0)
		if !leaf {
			leftChild = ChildAt(p, idx, n, t.Acc)
		}
		off := btpage.AllocCell(p, t.Acc.CellSize(leaf, value))
		t.Acc.WriteCell(p, off, key, value, leftChild, leaf)
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
	insertLC := uint32(0)
	if !leaf {
		insertLC = ChildAt(p, idx, n, t.Acc)
	}
	all[idx] = CellData{key, value, insertLC}

	oldRightmost := btpage.Rightmost(p)
	oldNext := btpage.NextLeaf(p)
	if !leaf {
		if idx == n {
			oldRightmost = uint32(rightChild)
		} else {
			all[idx+1].LeftChild = uint32(rightChild)
		}
	}

	mid := (n + 1) / 2
	promoted := all[mid]

	newID, _ := t.Pg.Allocate()
	right := new(pager.Page)

	// Reinitialize both pages FIRST
	btpage.InitPage(p, pageType)
	btpage.InitPage(right, pageType)

	if leaf {

		// LEFT
		for i := 0; i < mid; i++ {
			t.AppendCell(p, all[i].Key, all[i].Value, all[i].LeftChild)
		}

		// RIGHT (copy-up semantics)
		start := mid
		if !t.Acc.CopyUpLeaves() {
			start = mid + 1
		}
		for i := start; i <= n; i++ {
			t.AppendCell(right, all[i].Key, all[i].Value, all[i].LeftChild)
		}

		// Link leaves using the previous next pointer (not rightmost).
		t.Acc.LinkLeaves(p, right, uint32(newID), oldNext)

	} else {

		// LEFT keeps 0..mid-1
		for i := 0; i < mid; i++ {
			t.AppendCell(p, all[i].Key, all[i].Value, all[i].LeftChild)
		}
		btpage.SetRightmost(p, all[mid].LeftChild)

		// RIGHT keeps mid+1..n
		for i := mid + 1; i <= n; i++ {
			t.AppendCell(right, all[i].Key, all[i].Value, all[i].LeftChild)
		}
		btpage.SetRightmost(right, oldRightmost)
	}

	_ = t.Pg.Write(id, p)
	_ = t.Pg.Write(newID, right)

	// B+ tree copy-up uses separator keys only (no value).
	promotedVal := promoted.Value
	if leaf && t.Acc.CopyUpLeaves() {
		promotedVal = nil
	}
	return promoted.Key, promotedVal, newID, true, nil
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

func (t *Tree) Print(name string) {
	dotPath := fmt.Sprintf("results/%s.dot", name)
	pngPath := fmt.Sprintf("results/%s.png", name)

	if err := t.ExportDOT(dotPath); err != nil {
		fmt.Println("DOT export error:", err)
		return
	}

	// Graphviz ausführen
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", pngPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Graphviz error: %v (Stelle sicher, dass 'dot' installiert ist)\n", err)
		return
	}

	fmt.Printf("Tree erfolgreich exportiert nach: %s\n", pngPath)
}
func (t *Tree) ExportDOT(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "digraph BTree {")
	// Layout and Global Styling
	fmt.Fprintln(f, "  graph [ranksep=0.8, nodesep=0.5, bgcolor=\"#ffffff\", rankdir=TB];")
	fmt.Fprintln(f, "  node [shape=none, fontname=\"Helvetica\", fontsize=10];")
	fmt.Fprintln(f, "  edge [arrowsize=0.8, color=\"#444444\"];")

	nodeMap := make(map[uint64]string)
	var leafIDs []uint64
	var counter int

	var exportRec func(pageID uint64) string
	exportRec = func(pageID uint64) string {
		if name, ok := nodeMap[pageID]; ok {
			return name
		}
		nodeName := fmt.Sprintf("node%d", counter)
		counter++
		nodeMap[pageID] = nodeName

		p, err := t.Pg.Read(pageID)
		if err != nil {
			return nodeName
		}
		numCells := btpage.NumCells(p)
		leaf := isLeaf(p)

		// Calculate Fill Percentage (Total 4096 bytes)
		free := btpage.FreeSpace(p, numCells)
		usedPct := 100 - (float64(free) / 4096.0 * 100.0)

		if leaf {
			nextID := btpage.NextLeaf(p)
			// LEAF NODE: Green Header
			label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
				<TR><TD COLSPAN="2" BGCOLOR="#D5E8D4"><B>PAGE %d (LEAF)</B><BR/><FONT POINT-SIZE="8">Fill: %.1f%%</FONT></TD></TR>
				<TR><TD PORT="keys" BGCOLOR="#F5F5F5" ALIGN="LEFT">`, pageID, usedPct)

			for i := 0; i < numCells; i++ {
				k, v, _ := t.Acc.ReadCell(p, i, true)
				preview := ""
				if len(v) > 0 {
					pText := string(v)
					if len(pText) > 3 {
						pText = pText[:3] + ".."
					}
					preview = fmt.Sprintf(" <FONT COLOR='#666666'>[%s]</FONT>", pText)
				}
				label += fmt.Sprintf("<B>%d</B>%s<BR/>", k, preview)
			}

			nextLabel := "NULL"
			if nextID != 0 && nextID != 0xFFFFFFFF {
				nextLabel = fmt.Sprintf("%d", nextID)
			}
			label += fmt.Sprintf(`</TD><TD PORT="next" BGCOLOR="#E1F5FE" VALIGN="MIDDLE">Next: %s</TD></TR></TABLE>>`, nextLabel)

			fmt.Fprintf(f, "  %s [label=%s];\n", nodeName, label)
			leafIDs = append(leafIDs, pageID)
		} else {
			// INTERNAL NODE: Blue Header
			label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
				<TR><TD COLSPAN="%d" BGCOLOR="#DAE8FC"><B>PAGE %d (INTERNAL)</B><BR/><FONT POINT-SIZE="8">Fill: %.1f%%</FONT></TD></TR><TR>`, (numCells*2)+1, pageID, usedPct)

			for i := 0; i < numCells; i++ {
				k, v, leftChild := t.Acc.ReadCell(p, i, false)

				valPreview := ""
				if v != nil && len(v) > 0 {
					pText := string(v)
					if len(pText) > 3 {
						pText = pText[:3] + ".."
					}
					valPreview = fmt.Sprintf("<BR/><FONT POINT-SIZE='7' COLOR='#444444'>[%s]</FONT>", pText)
				}

				label += fmt.Sprintf(`<TD PORT="f%d" BGCOLOR="#E1F5FE">P:%d</TD><TD BGCOLOR="#FFFFFF"><B>%d</B>%s</TD>`, i, leftChild, k, valPreview)
			}
			rightID := btpage.Rightmost(p)
			label += fmt.Sprintf(`<TD PORT="f%d" BGCOLOR="#E1F5FE">P:%d</TD></TR></TABLE>>`, numCells, rightID)

			fmt.Fprintf(f, "  %s [label=%s];\n", nodeName, label)

			// Draw edges to children
			for i := 0; i < numCells; i++ {
				_, _, leftChild := t.Acc.ReadCell(p, i, false)
				childName := exportRec(uint64(leftChild))
				fmt.Fprintf(f, "  %s:f%d -> %s;\n", nodeName, i, childName)
			}
			childName := exportRec(uint64(rightID))
			fmt.Fprintf(f, "  %s:f%d -> %s;\n", nodeName, numCells, childName)
		}
		return nodeName
	}

	exportRec(uint64(t.RootID))

	// Link leaves horizontally (B+ Tree logic)
	if t.Acc.CopyUpLeaves() && len(leafIDs) > 1 {
		fmt.Fprintln(f, "  { rank=same;")
		for _, id := range leafIDs {
			fmt.Fprintf(f, "    %s;\n", nodeMap[id])
		}
		fmt.Fprintln(f, "  }")

		for _, id := range leafIDs {
			p, _ := t.Pg.Read(id)
			nextID := uint64(btpage.NextLeaf(p))
			if nextID != 0 && nextID != 0xFFFFFFFF {
				if targetName, ok := nodeMap[nextID]; ok {
					fmt.Fprintf(f, "  %s:next -> %s [style=dashed, color=\"#03A9F4\", constraint=false, tailclip=false];\n", nodeMap[id], targetName)
				}
			}
		}
	}

	fmt.Fprintln(f, "}")
	return nil
}
