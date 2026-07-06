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
// Package sampling provides log sampling functionality
// Author: Admilson B. F. Cossa

package sampling

// GetRate returns the current sampling rate as a percentage (0.0 to 1.0)
func (s *SamplingManager) GetRate() float64 {
	switch s.config.Strategy {
	case SampleByCount:
		if s.config.SamplingDenominator <= 0 {
			return 1.0 // 100% sampling
		}
		return 1.0 / float64(s.config.SamplingDenominator)
	case SampleByTime:
		// For time-based sampling, return rate based on max per window
		// This is an approximation since actual rate depends on log volume
		if s.config.MaxPerWindow <= 0 {
			return 1.0
		}
		return float64(s.config.MaxPerWindow) / 100.0 // Assume 100 logs/sec baseline
	case SampleByLevel:
		// For level-based sampling, return the average rate across all levels
		totalRate := 0.0
		count := 0
		for _, denom := range s.config.LevelSampling {
			if denom > 0 {
				totalRate += 1.0 / float64(denom)
				count++
			}
		}
		if count == 0 {
			return 1.0 / float64(s.config.SamplingDenominator)
		}
		return totalRate / float64(count)
	default:
		return 1.0
	}
}

// SetRate updates the sampling rate (0.0 to 1.0)
func (s *SamplingManager) SetRate(rate float64) {
	if rate <= 0.0 {
		rate = 0.01 // Minimum 1% sampling
	} else if rate > 1.0 {
		rate = 1.0 // Maximum 100% sampling
	}

	switch s.config.Strategy {
	case SampleByCount:
		// Convert rate to denominator (e.g., 0.1 -> 10, 0.5 -> 2, 1.0 -> 1)
		s.config.SamplingDenominator = int(1.0 / rate)
		if s.config.SamplingDenominator < 1 {
			s.config.SamplingDenominator = 1
		}
	case SampleByTime:
		// For time-based, adjust max per window proportionally
		s.config.MaxPerWindow = int(rate * 100.0) // Scale to reasonable range
		if s.config.MaxPerWindow < 1 {
			s.config.MaxPerWindow = 1
		}
	case SampleByLevel:
		// For level-based, adjust all level denominators proportionally
		oldRate := s.GetRate()
		if oldRate > 0 {
			scale := rate / oldRate
			for level, denom := range s.config.LevelSampling {
				if denom > 0 {
					s.config.LevelSampling[level] = int(float64(denom) / scale)
					if s.config.LevelSampling[level] < 1 {
						s.config.LevelSampling[level] = 1
					}
				}
			}
		}
	}
}
