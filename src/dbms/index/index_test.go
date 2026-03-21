package index_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/btree-query-bench/bmark/dbms/index"
	"github.com/btree-query-bench/bmark/dbms/index/bptree"
	"github.com/btree-query-bench/bmark/dbms/index/btree"
	"github.com/btree-query-bench/bmark/dbms/index/lsm"
)

func runIndexTests(t *testing.T, newIdx func(path string) (index.Index, error), name string) {
	t.Run(name+"/InsertAndGet", func(t *testing.T) {
		path := fmt.Sprintf("/tmp/idx_test_%s_ig", name)
		defer os.RemoveAll(path)
		defer os.RemoveAll(path + ".bt")
		defer os.RemoveAll(path + ".bpt")

		idx, err := newIdx(path)
		if err != nil {
			t.Fatal(err)
		}
		defer idx.Close()

		key := int64(42)
		val := []byte("hello world")

		if err := idx.Insert(key, val); err != nil {
			t.Errorf("Insert failed: %v", err)
		}

		got, err := idx.Get(key)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if !bytes.Equal(got, val) {
			t.Errorf("Got %s, want %s", got, val)
		}

		// Test not found
		got, err = idx.Get(999)
		if err != nil {
			t.Errorf("Get(999) failed: %v", err)
		}
		if got != nil {
			t.Errorf("Got %s for missing key, want nil", got)
		}
	})

	t.Run(name+"/Overwrite", func(t *testing.T) {
		path := fmt.Sprintf("/tmp/idx_test_%s_ov", name)
		defer os.RemoveAll(path)
		defer os.RemoveAll(path + ".bt")
		defer os.RemoveAll(path + ".bpt")

		idx, err := newIdx(path)
		if err != nil {
			t.Fatal(err)
		}
		defer idx.Close()

		key := int64(10)
		v1 := []byte("value 1")
		v2 := []byte("value 2 (longer)")

		_ = idx.Insert(key, v1)
		_ = idx.Insert(key, v2)

		got, _ := idx.Get(key)
		if !bytes.Equal(got, v2) {
			t.Errorf("Overwrite failed: got %s, want %s", got, v2)
		}
	})

	t.Run(name+"/RangeQuery", func(t *testing.T) {
		path := fmt.Sprintf("/tmp/idx_test_%s_rq", name)
		defer os.RemoveAll(path)
		defer os.RemoveAll(path + ".bt")
		defer os.RemoveAll(path + ".bpt")

		idx, err := newIdx(path)
		if err != nil {
			t.Fatal(err)
		}
		defer idx.Close()

		// Insert 10, 20, 30, 40, 50
		for i := 1; i <= 5; i++ {
			k := int64(i * 10)
			v := []byte(fmt.Sprintf("val%d", k))
			if err := idx.Insert(k, v); err != nil {
				t.Fatal(err)
			}
		}

		// Range [15, 35] -> should return 20, 30
		it, err := idx.Range(15, 35)
		if err != nil {
			t.Fatal(err)
		}
		defer it.Close()

		expected := []int64{20, 30}
		count := 0
		for it.Next() {
			if count >= len(expected) {
				t.Errorf("Too many results from range query")
				break
			}
			if it.Key() != expected[count] {
				t.Errorf("Range at index %d: got key %d, want %d", count, it.Key(), expected[count])
			}
			count++
		}
		if count != len(expected) {
			t.Errorf("Range query returned %d items, want %d", count, len(expected))
		}
		if it.Error() != nil {
			t.Errorf("Iterator error: %v", it.Error())
		}
	})

	t.Run(name+"/SplitNodes", func(t *testing.T) {
		path := fmt.Sprintf("/tmp/idx_test_%s_split", name)
		defer os.RemoveAll(path)
		defer os.RemoveAll(path + ".bt")
		defer os.RemoveAll(path + ".bpt")

		// Small cache to force disk activity and use small page limits
		idx, err := newIdx(path)
		if err != nil {
			t.Fatal(err)
		}
		defer idx.Close()

		// Insert 500 keys to force multiple splits and tree growth
		n := 500
		for i := 1; i <= n; i++ {
			k := int64(i)
			v := bytes.Repeat([]byte{byte(i % 256)}, 100) // 100 byte values
			if err := idx.Insert(k, v); err != nil {
				t.Fatalf("Insert %d failed: %v", k, err)
			}
		}

		// Verify all 500 keys
		for i := 1; i <= n; i++ {
			k := int64(i)
			got, err := idx.Get(k)
			if err != nil {
				t.Errorf("Get %d failed: %v", k, err)
			}
			if len(got) != 100 {
				t.Errorf("Get %d: length mismatch, got %d want 100", k, len(got))
			}
		}

		// Test range over the whole set
		it, err := idx.Range(1, int64(n))
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for it.Next() {
			count++
		}
		if count != n {
			t.Errorf("Full range scan got %d, want %d", count, n)
		}
		it.Close()
	})
}

func TestBTree(t *testing.T) {
	runIndexTests(t, func(path string) (index.Index, error) {
		return btree.Open(path, 10)
	}, "BTree")
}

func TestBPTree(t *testing.T) {
	runIndexTests(t, func(path string) (index.Index, error) {
		return bptree.Open(path, 10)
	}, "BPTree")
}

func TestLSM(t *testing.T) {
	runIndexTests(t, func(path string) (index.Index, error) {
		return lsm.Open(path)
	}, "LSM")
}
