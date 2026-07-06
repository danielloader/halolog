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
// Package types provides core type definitions
// Author: Admilson B. F. Cossa

package types

// Style bitflags for text styling (High-performance: O(1) bitwise operations)
const (
	StyleBold      uint8 = 1 << 0 // 0x01
	StyleItalic    uint8 = 1 << 1 // 0x02
	StyleUnderline uint8 = 1 << 2 // 0x04
	StyleHighlight uint8 = 1 << 3 // 0x08
	StyleDim       uint8 = 1 << 4 // 0x10
	StyleBlink     uint8 = 1 << 5 // 0x20
	StyleReverse   uint8 = 1 << 6 // 0x40
	StyleHidden    uint8 = 1 << 7 // 0x80
)

// HasStyle checks if a style flag is set (O(1) bitwise AND)
func HasStyle(style uint8, flag uint8) bool {
	return (style & flag) != 0
}

// AddStyle adds a style flag (O(1) bitwise OR)
func AddStyle(style uint8, flag uint8) uint8 {
	return style | flag
}

// RemoveStyle removes a style flag (O(1) bitwise AND NOT)
func RemoveStyle(style uint8, flag uint8) uint8 {
	return style &^ flag
}

// ToggleStyle toggles a style flag (O(1) bitwise XOR)
func ToggleStyle(style uint8, flag uint8) uint8 {
	return style ^ flag
}
