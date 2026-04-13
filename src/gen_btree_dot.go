package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btree-query-bench/bmark/dbms/index/btree"
)

func main() {
	path := "test_btree"
	// Use a small page size to trigger splits more easily
	pageSize := uint32(512)
	cachePages := 10

	// Remove old file if it exists
	os.Remove(path + ".bt")

	t, err := btree.Open(path, cachePages, pageSize)
	if err != nil {
		log.Fatalf("failed to open btree: %v", err)
	}
	defer t.Close()

	for i := 1; i <= 100; i++ {
		key := int64(i)
		val := []byte(fmt.Sprintf("value-%d", i))
		if err := t.Insert(key, val); err != nil {
			log.Fatalf("failed to insert key %d: %v", i, err)
		}
	}

	dotFile := "btree.dot"
	if err := t.ExportDOT(dotFile); err != nil {
		log.Fatalf("failed to export DOT: %v", err)
	}

	fmt.Printf("B-tree DOT file generated: %s\n", dotFile)
}
