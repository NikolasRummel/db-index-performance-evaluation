package lsmtree

import "hash/fnv"

type BloomFilter struct {
	bits []bool
	m    uint32
	k    int
}

func NewBloom(size int, k int) *BloomFilter {
	return &BloomFilter{
		bits: make([]bool, size),
		m:    uint32(size),
		k:    k,
	}
}

func (b *BloomFilter) getHashes(key int64) []uint32 {
	hashes := make([]uint32, b.k)
	h := fnv.New32a()
	// Convert int64 to bytes
	keyBytes := []byte{
		byte(key), byte(key >> 8), byte(key >> 16), byte(key >> 24),
		byte(key >> 32), byte(key >> 40), byte(key >> 48), byte(key >> 56),
	}
	for i := 0; i < b.k; i++ {
		h.Write([]byte{byte(i)}) // Salt for different hashes
		h.Write(keyBytes)
		hashes[i] = h.Sum32() % b.m
		h.Reset()
	}
	return hashes
}

func (b *BloomFilter) Add(key int64) {
	for _, h := range b.getHashes(key) {
		b.bits[h] = true
	}
}

func (b *BloomFilter) Test(key int64) bool {
	for _, h := range b.getHashes(key) {
		if !b.bits[h] {
			return false // Definitely not there
		}
	}
	return true // Might be there
}
