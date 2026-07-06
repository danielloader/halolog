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
// Package utils provides utility functions
// Author: Admilson B. F. Cossa

package utils

import "strconv"

func appendIntAny(buf []byte, v interface{}) []byte {
	switch x := v.(type) {
	case int:
		return strconv.AppendInt(buf, int64(x), 10)
	case int8:
		return strconv.AppendInt(buf, int64(x), 10)
	case int16:
		return strconv.AppendInt(buf, int64(x), 10)
	case int32:
		return strconv.AppendInt(buf, int64(x), 10)
	case int64:
		return strconv.AppendInt(buf, x, 10)
	}
	return buf
}
