package main

func consumeSuccesses(config *ConsumerConfig) {
	foundCount := 0
	var earliestID int64
	for instanceId := range config.SuccessChannel {
		if !config.GapMode {
			// If we are not in gap mode, we should not do anything
			continue
		}
		foundCount++
		if instanceId < earliestID || earliestID == 0 {
			earliestID = instanceId
		}
		if foundCount >= 100 {
			exitGapModeAlert(foundCount, earliestID)
			config.GapMode = false
			foundCount = 0
			earliestID = 0
		}
	}
}
