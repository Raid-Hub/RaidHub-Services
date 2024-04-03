package main

const (
	startBuffer int64 = 10_000

	minWorkers = 5
	maxWorkers = 200

	periodLength       = 25_000
	circularBufferSize = 50
	errorBufferSize    = 25

	gapModeWorkers = 500

	numMissesForWarning = 7
	numMisses           = 10
	retryDelayTime      = 5000
)
