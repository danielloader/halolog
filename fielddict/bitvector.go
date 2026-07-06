// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package fielddict provides field dictionary management
// Author: Admilson B. F. Cossa

package fielddict

import "sync"

// BitVector provides efficient bit operations for field optimization
type BitVector struct {
	mu   sync.RWMutex
	bits []uint64
	size int
}

// NewBitVector creates a new bit vector with the specified size
func NewBitVector(size int) *BitVector {
	return &BitVector{
		bits: make([]uint64, (size+63)/64),
		size: size,
	}
}

// Set sets the bit at the specified position
func (bv *BitVector) Set(pos int) {
	if pos >= 0 && pos < bv.size {
		bv.mu.Lock()
		bv.bits[pos/64] |= 1 << (pos % 64)
		bv.mu.Unlock()
	}
}

// Get returns true if the bit at the specified position is set
func (bv *BitVector) Get(pos int) bool {
	if pos >= 0 && pos < bv.size {
		bv.mu.RLock()
		defer bv.mu.RUnlock()
		return bv.bits[pos/64]&(1<<(pos%64)) != 0
	}
	return false
}

// Clear clears the bit at the specified position
func (bv *BitVector) Clear(pos int) {
	if pos >= 0 && pos < bv.size {
		bv.mu.Lock()
		bv.bits[pos/64] &^= 1 << (pos % 64)
		bv.mu.Unlock()
	}
}

// Size returns the size of the bit vector
func (bv *BitVector) Size() int {
	return bv.size
}

// Count returns the number of set bits
func (bv *BitVector) Count() int {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	count := 0
	for _, word := range bv.bits {
		count += popcount(word)
	}
	return count
}

// popcount counts the number of set bits in a 64-bit word
func popcount(x uint64) int {
	// Hamming weight algorithm
	x = (x & 0x5555555555555555) + ((x >> 1) & 0x5555555555555555)
	x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
	x = (x & 0x0F0F0F0F0F0F0F0F) + ((x >> 4) & 0x0F0F0F0F0F0F0F0F)
	x = (x & 0x00FF00FF00FF00FF) + ((x >> 8) & 0x00FF00FF00FF00FF)
	x = (x & 0x0000FFFF0000FFFF) + ((x >> 16) & 0x0000FFFF0000FFFF)
	x = (x & 0x00000000FFFFFFFF) + ((x >> 32) & 0x00000000FFFFFFFF)
	return int(x)
}
