package main

const (
	startBuffer int64 = 15_000

	minWorkers = 5
	maxWorkers = 200

	periodLength       = 25_000
	circularBufferSize = 50
	errorBufferSize    = 25

	gapModeWorkers              = 500
	decreaseBelow               = 40
	increaseAbove               = 42
	multiplicativeIncreaseAbove = 80

	numMissesForWarning = 5
	numMisses           = 10
	retryDelayTime      = 5000
)
