package main

type ConsumerConfig struct {
	LatestId        int64
	GapMode         bool
	FailuresChannel chan int64
	SuccessChannel  chan int64
}

type WorkerResult struct {
	Lag       []float64
	NotFounds int
}
