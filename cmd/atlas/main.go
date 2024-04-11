package main

import (
	"database/sql"
	"flag"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/joho/godotenv"

	"raidhub/shared/async"
	"raidhub/shared/monitoring"
	"raidhub/shared/postgres"
)

var (
	numWorkers   = flag.Int("workers", 50, "number of workers to spawn at the start")
	buffer       = flag.Int64("buffer", 10_000, "number of ids to start behind last added")
	workers      = 0
	periodLength = 50_000
)

// bin/atlas <numWorkersStart> <offset>
func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	workers = *numWorkers
	if *buffer < 0 || workers <= 0 || workers > maxWorkers {
		log.Fatalln("Invalid flags")
	}

	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}
	defer db.Close()

	instanceId, err := postgres.GetLatestInstanceId(db, *buffer)
	if err != nil {
		log.Fatalf("Error getting latest instance id: %s", err)
	}

	monitoring.RegisterPrometheus(8080)

	run(instanceId, db)

}

func run(latestId int64, db *sql.DB) {
	defer func() {
		if r := recover(); r != nil {
			handlePanic(r)
		}
	}()

	conn, err := async.Init()
	if err != nil {
		log.Fatalf("Failed to create connection: %s", err)
	}
	defer async.Cleanup()

	rabbitChannel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %s", err)
	}
	defer rabbitChannel.Close()

	consumerConfig := ConsumerConfig{
		LatestId:         latestId,
		GapMode:          false,
		FailuresChannel:  make(chan int64),
		SuccessChannel:   make(chan int64),
		MalformedChannel: make(chan int64),
		RabbitChannel:    rabbitChannel,
	}

	sendStartUpAlert()

	// Start a goroutine to consume failures from the channel
	go consumeFailures(&consumerConfig)

	// Start a goroutine to consume found PGCRs from the gap mode channel
	go consumeSuccesses(&consumerConfig)

	// Start a goroutine to consume malformed PGCRs
	go malformedWorker(consumerConfig.MalformedChannel, consumerConfig.RabbitChannel, db)

	for {
		if !consumerConfig.GapMode {
			workers = spawnWorkers(workers, db, &consumerConfig)
		} else {
			spawnGapModeWorkers(db, &consumerConfig)
			// When exiting gap mode, we should increase the number of workers to catch up
			workers = maxWorkers
		}
	}

}

func spawnWorkers(countWorkers int, db *sql.DB, consumerConfig *ConsumerConfig) int {
	var wg sync.WaitGroup
	ids := make(chan int64, 5)

	logWorkersStarting(countWorkers, periodLength, consumerConfig.LatestId)

	// When each worker finishes, it will send its info onto these channels
	resultsChannel := make(chan *WorkerResult, countWorkers)

	for i := 0; i < countWorkers; i++ {
		wg.Add(1)
		go Worker(&wg, ids, resultsChannel, consumerConfig.FailuresChannel, consumerConfig.MalformedChannel, consumerConfig.RabbitChannel, db)
	}

	// Pass IDs to workers
	for i := 0; i < periodLength; i++ {
		// If we are in gap mode is entered, we should stop passing IDs to workers
		if consumerConfig.GapMode {
			break
		}
		consumerConfig.LatestId++
		ids <- consumerConfig.LatestId
	}

	close(ids)
	wg.Wait()
	close(resultsChannel)

	var lags []float64
	var notFound int
	for result := range resultsChannel {
		lags = append(lags, result.Lag...)
		notFound += result.NotFounds
	}

	arrSlice := lags[:]
	sort.Float64s(arrSlice)
	n := len(arrSlice)

	var medianLag float64
	if n%2 == 1 {
		medianLag = arrSlice[n/2]
	} else if n > 0 {
		// Even number of elements
		medianLag = (arrSlice[n/2-1] + arrSlice[n/2]) / 2.0
	}
	fractionNotFound := float64(notFound) / float64(periodLength)

	logIntervalState(medianLag, countWorkers, fractionNotFound*100)

	newWorkers := 0
	if fractionNotFound == 0 {
		// how much we expect to get catch up
		periodLength = countWorkers * 4 * (int(math.Ceil(medianLag)) - 30) / 3
		if periodLength < 10_000 {
			periodLength = 10_000
		}
		// If we aren't getting 404's, just spike the workers up to ensure we catch up to live ASAP
		newWorkers = int(math.Ceil(float64(countWorkers) * (1 + float64(medianLag-30)/100)))

	} else {
		decreaseFraction := (retryDelayTime / 800 * (fractionNotFound - 0.032)) // do not let workers go below 3.2%
		if decreaseFraction > 0.8 {
			decreaseFraction = 0.8
		}
		// Adjust number of workers for the next period
		newWorkers = int(math.Round(float64(countWorkers) - decreaseFraction*float64(countWorkers)))
		periodLength = 500 * newWorkers
	}

	if newWorkers > maxWorkers {
		newWorkers = maxWorkers
	} else if newWorkers < minWorkers {
		newWorkers = minWorkers
	}
	return newWorkers
}

func spawnGapModeWorkers(db *sql.DB, consumerConfig *ConsumerConfig) {
	var wg sync.WaitGroup
	ids := make(chan int64, 25)

	for i := 0; i < gapModeWorkers; i++ {
		wg.Add(1)
		go gapModeWorker(&wg, ids, consumerConfig.SuccessChannel, db, consumerConfig.RabbitChannel)
	}

	misses := 0
	// Pass IDs to workers, but only if we are in gap mode
	for {
		if !consumerConfig.GapMode {
			break
		}
		if misses > 100_000 {
			gapModeFailureAlert()
		}
		consumerConfig.LatestId++
		ids <- consumerConfig.LatestId
	}

	close(ids)
	wg.Wait()
}
