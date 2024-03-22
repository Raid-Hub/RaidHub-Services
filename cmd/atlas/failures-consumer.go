package main

import (
	"sort"
)

func consumeFailures(config *ConsumerConfig) {
	var failures []int64
	for failureID := range config.FailuresChannel {
		// Append the new failureID to the slice
		failures = append(failures, failureID)

		if len(failures) > errorBufferSize {
			// Trim the slice not not exceed the errorBufferSize
			failures = failures[1:]
			// Order the slice
			sort.Slice(failures, func(i, j int) bool {
				return failures[i] < failures[j]
			})

			if config.GapMode {
				// If we already in gap mode, we don't need to check for density
				continue
			}

			variation := failures[len(failures)-1] - failures[0]
			density := float64(errorBufferSize) / float64(variation)

			if density >= 0.2 {
				config.GapMode = true
				enterGapModeAlert(variation, failures[0], density)
			}
		}
	}
}
