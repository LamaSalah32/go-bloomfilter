package bloomfilter

import (
	"errors"
	"math"
	"math/bits"

	"github.com/cespare/xxhash/v2"
)

type BloomFilter struct {
	m, k uint64
	bit  []uint64
}

func computeHashes(data []byte) (uint64, uint64) {
	h1 := xxhash.Sum64(data)
	h2 := h1 ^ 0x8E3C5B2F1A0D9E74
	h2 = h2 * 0x9E3779B97F4A7C15
	h2 ^= h2 >> 33

	return h1, h2
}

func getIndices(h1, h2, i, m uint64) uint64 {
	h := h1 + i*h2
	return h % m
}

func New(m, k uint64) *BloomFilter {
	return &BloomFilter{
		m,
		k,
		make([]uint64, (m+63)/64),
	}
}

func CalcParamsWithFPR(n uint64, p float64) (m, k uint64) {
	m = uint64(math.Ceil(-1 * float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
	k = uint64(math.Ceil(math.Log(2) * float64(m) / float64(n)))
	return
}

func NewWithFPR(n uint64, p float64) *BloomFilter {
	m, k := CalcParamsWithFPR(n, p)
	return New(m, k)
}

func (bf *BloomFilter) Add(data []byte) {
	h1, h2 := computeHashes(data)

	for i := uint64(0); i < bf.k; i++ {
		bit := getIndices(h1, h2, i, bf.m)
		bf.bit[bit/64] |= uint64(1) << uint64(bit%64)
	}
}

func (bf *BloomFilter) Contains(data []byte) bool {
	h1, h2 := computeHashes(data)

	for i := uint64(0); i < bf.k; i++ {
		bit := getIndices(h1, h2, i, bf.m)
		if (uint64(1)<<uint64(bit%64))&(bf.bit[bit/64]) == 0 {
			return false
		}
	}

	return true
}

func (bf *BloomFilter) Clear() *BloomFilter {
	bf.bit = make([]uint64, (bf.m+63)/64)
	return bf
}

func (bf1 *BloomFilter) Union(bf2 *BloomFilter) (err error) {
	if bf1.m != bf2.m {
		err = errors.New("bloom filters have mismatched sizes")
	}

	if bf1.k != bf2.k {
		err = errors.New("bloom filters have mismatched hash counts")
	}

	for i, _ := range bf1.bit {
		bf1.bit[i] |= bf2.bit[i]
	}

	return
}

func (bf1 *BloomFilter) Intersection(bf2 *BloomFilter) (err error) {
	if bf1.m != bf2.m {
		err = errors.New("bloom filters have mismatched sizes")
	}

	if bf1.k != bf2.k {
		err = errors.New("bloom filters have mismatched hash counts")
	}

	for i, _ := range bf1.bit {
		bf1.bit[i] &= bf2.bit[i]
	}

	return
}

func (bf1 *BloomFilter) Subset(bf2 *BloomFilter) (err error, isSubset bool) {
	if bf1.m != bf2.m {
		err = errors.New("bloom filters have mismatched sizes")
	}

	if bf1.k != bf2.k {
		err = errors.New("bloom filters have mismatched hash counts")
	}

	isSubset = true
	for i := 0; i < len(bf1.bit); i++ {
		if bf1.bit[i]&bf2.bit[i] != bf2.bit[i] {
			isSubset = false
			break
		}
	}

	return
}

func CountZeroBits(bf *BloomFilter) uint64 {
	cnt := uint64(0)
	for _, v := range bf.bit {
		cnt += uint64(bits.OnesCount64(v))
	}

	return bf.m - cnt
}

func (bf *BloomFilter) ApproximateCardinality() (n int64) {
	m := bf.m
	k := bf.k
	x := CountZeroBits(bf)

	n = int64(-float64(m) / float64(k) * math.Log(float64(x)/float64(m)))
	return
}
