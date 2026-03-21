package bench

import (
	"encoding/binary"
	"math/rand"
	"slices"
)

type Dataset struct {
	Keys   []int64
	Values [][]byte
	rng    *rand.Rand
}

func NewDataset(n int, valueSize int, seed int64) *Dataset {
	rng := rand.New(rand.NewSource(seed))

	keys := make([]int64, n)
	for i := range keys {
		keys[i] = int64(i + 1)
	}
	rng.Shuffle(n, func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	values := make([][]byte, n)
	for i := range values {
		v := make([]byte, valueSize)
		binary.LittleEndian.PutUint64(v, uint64(keys[i]))
		rng.Read(v[8:])
		values[i] = v
	}

	return &Dataset{
		Keys:   keys,
		Values: values,
		rng:    rng,
	}
}

func (d *Dataset) SortedKeys() []int64 {
	sorted := make([]int64, len(d.Keys))
	copy(sorted, d.Keys)
	slices.Sort(sorted)
	return sorted
}

func (d *Dataset) RandomKeys(m int) []int64 {
	perm := d.rng.Perm(len(d.Keys))
	out := make([]int64, m)
	for i := range out {
		out[i] = d.Keys[perm[i]]
	}
	return out
}
