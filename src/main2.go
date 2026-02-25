package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index" // Import your interface package
	"github.com/btree-query-bench/bmark/dbms/index/bptree"
	"github.com/btree-query-bench/bmark/dbms/index/btree"
)

func main() {
	// Erstelle den Ergebnisordner, falls er nicht existiert
	_ = os.Mkdir("results", 0755)

	// Test B-Tree
	fmt.Println("--- Testing B-Tree ---")
	bt, _ := btree.Open("test", 10)
	runTest(bt, "btree_result") // Wir fügen einen Namen hinzu

	// Test B+ Tree
	fmt.Println("\n--- Testing B+ Tree ---")
	bpt, _ := bptree.Open("test", 10)
	runTest(bpt, "bptree_result")

	os.Remove("test.bt")
	os.Remove("test.bpt")
}
func runTest(idx index.Index, filename string) {
	defer idx.Close()

	fmt.Println("1. Stress Testing for Multi-Level Growth...")

	largeValue := make([]byte, 512)
	for i := range largeValue {
		largeValue[i] = 'X'
	}

	// Inserting 60 keys will definitely force a 3-level tree.
	// Each page holds ~7 cells.
	// ~9 leaf pages will be created, forcing the internal level to split.
	for k := int64(1); k <= 60; k++ {
		if err := idx.Insert(k, largeValue); err != nil {
			log.Fatalf("Insert failed for %d: %v", k, err)
		}
		if k%10 == 0 {
			fmt.Printf("Inserted %d keys... ", k)
		}
	}

	fmt.Println("\n2. Verifying Point Lookup...")
	val, _ := idx.Get(30)
	if len(val) != 512 {
		log.Fatalf("Value size mismatch. Expected 512, got %d", len(val))
	}
	fmt.Println("Lookup 30 OK.")

	fmt.Println("3. Deep Range Scan...")
	// Scan across multiple leaf pages
	it, _ := idx.Range(5, 55)
	count := 0
	for it.Next() {
		count++
	}
	it.Close()
	fmt.Printf("Range scan OK. Found %d keys.\n", count)

	// 4. Print the tree structure (this will now be a much larger image)
	if t, ok := idx.(interface{ Print(string) }); ok {
		t.Print(filename)
	}
}
