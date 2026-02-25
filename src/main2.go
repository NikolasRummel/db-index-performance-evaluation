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
	os.Remove("test.bt")
	os.Remove("test.bpt")

	// Test B-Tree
	fmt.Println("--- Testing B-Tree ---")
	bt, _ := btree.Open("test", 10)
	runTest(bt)

	// Test B+ Tree
	fmt.Println("\n--- Testing B+ Tree ---")
	bpt, _ := bptree.Open("test", 10)
	runTest(bpt)
}

func runTest(idx index.Index) {
	defer idx.Close()

	fmt.Println("1. Basic Functional Tests...")
	// Insert out of order
	keys := []int64{100, 50, 150, 25, 75, 125, 175}
	for _, k := range keys {
		if err := idx.Insert(k, []byte(fmt.Sprintf("data-%d", k))); err != nil {
			log.Fatalf("Insert failed for %d: %v", k, err)
		}
	}

	// Update existing key
	if err := idx.Insert(75, []byte("updated-75")); err != nil {
		log.Fatalf("Update failed: %v", err)
	}

	// Verify updates and point lookups
	val, _ := idx.Get(75)
	if string(val) != "updated-75" {
		log.Fatalf("Update check failed. Got: %s", string(val))
	}

	val, _ = idx.Get(999) // Non-existent
	if val != nil {
		log.Fatalf("Expected nil for non-existent key, got %v", val)
	}

	fmt.Println("2. Range Scan Edge Cases...")
	// Test cases: [start, end]
	scanTests := []struct {
		start, end int64
		expected   int // count of items
	}{
		{25, 175, 7},  // Full range
		{60, 130, 3},  // Middle range (75, 100, 125)
		{200, 300, 0}, // Out of bounds high
		{0, 10, 0},    // Out of bounds low
		{50, 50, 1},   // Single point range
	}

	for _, st := range scanTests {
		it, err := idx.Range(st.start, st.end)
		if err != nil {
			log.Fatal(err)
		}
		count := 0
		for it.Next() {
			count++
		}
		if count != st.expected {
			fmt.Printf("Range [%d, %d] FAILED: expected %d items, got %d\n", st.start, st.end, st.expected, count)
		} else {
			fmt.Printf("Range [%d, %d] OK (%d items)\n", st.start, st.end, count)
		}
		it.Close()
	}

	fmt.Println("3. Split & Continuity Stress Test...")
	// Insert 500 keys to force multiple levels of splits.
	// This tests if Copy-Up (B+) and Push-Up (B) preserve the tree structure correctly.
	for i := int64(1000); i < 1500; i++ {
		idx.Insert(i, []byte("val"))
	}

	// Range scan across the split boundaries
	it, _ := idx.Range(1490, 1510)
	last := int64(0)
	for it.Next() {
		last = it.Key()
	}
	if last != 1499 {
		log.Fatalf("Stress range failed. Last key should be 1499, got %d", last)
	}
	it.Close()
	fmt.Println("Stress test passed.")
}
