package main

import (
	"math/rand"

	"github.com/btree-query-bench/bmark/index"
)

type WorkloadType string

const (
	OLTP      WorkloadType = "OLTP (90/10)"
	OLAP      WorkloadType = "OLAP (10/90)"
	Reporting WorkloadType = "Reporting (Range)"
)

// ExecuteWorkload runs a mixed distribution of ops
func ExecuteWorkload(idx index.Index, wType WorkloadType, ops int) {
	for i := 0; i < ops; i++ {
		choice := rand.Intn(100)
		key := int64(rand.Intn(ops))

		switch wType {
		case OLTP:
			if choice < 90 {
				_, _ = idx.Get(key)
			} else {
				idx.Insert(key, []byte("x"))
			}
		case OLAP:
			if choice < 10 {
				_, _ = idx.Get(key)
			} else {
				idx.Insert(key, []byte("x"))
			}
		case Reporting:
			it, _ := idx.Range(key, key+100)
			if it != nil {
				for it.Next() {
				}
			}
		}
	}
}
