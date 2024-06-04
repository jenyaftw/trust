package utils

import (
	"crypto/rand"
	"math"
	"math/big"
)

func GenerateRandomId() uint64 {
	rint, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return rint.Uint64()
}

func GetLastBit(n int) int {
	if n == 0 {
		return 0
	}

	for n > 1 {
		n >>= 1
	}

	return n
}

func GetFirstBit(n, max int) int {
	bitCount := GetBitCount(n)
	maxBitCount := GetBitCount(max)
	if maxBitCount > bitCount {
		return 0
	}
	return (n & (1 << (bitCount - 1))) >> (bitCount - 1)
}

func GetBitCount(n int) int {
	bits := 0
	for n > 0 {
		bits++
		n >>= 1
	}
	return bits
}

func GetMasks(bitCount int) (int, int, int) {
	allMask := 0b1
	lastMask := 0b1
	firstMask := 0b1

	for i := 1; i < bitCount; i++ {
		allMask = (allMask << 1) | 1
		lastMask = lastMask >> 1
		firstMask = firstMask << 1
	}

	return allMask, lastMask, firstMask
}
