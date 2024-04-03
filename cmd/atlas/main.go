package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/joho/godotenv"

	"raidhub/shared/monitoring"
	"raidhub/shared/postgres"
)

// ./atlas <numWorkersStart>
func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) < 2 {
		fmt.Println("Please specify the number of workers to start with.")
		return
	}

	numWorkersStart, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing integer for numConcurrentFiles: %s", err)
	}

	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}
	defer db.Close()

	instanceId, err := postgres.GetLatestInstanceId(db, startBuffer)
	if err != nil {
		log.Fatalf("Error getting latest instance id: %s", err)
	}

	monitoring.RegisterPrometheus(8080)

	run(numWorkersStart, instanceId, db)

}

func run(numWorkers int, latestId int64, db *sql.DB) {
	defer func() {
		if r := recover(); r != nil {
			handlePanic(r)
		}
	}()

	consumerConfig := ConsumerConfig{
		LatestId:         latestId,
		GapMode:          false,
		FailuresChannel:  make(chan int64),
		SuccessChannel:   make(chan int64),
		MalformedChannel: make(chan int64),
	}

	sendStartUpAlert()

	// Start a goroutine to consume failures from the channel
	go consumeFailures(&consumerConfig)

	// Start a goroutine to consume found PGCRs from the gap mode channel
	go consumeSuccesses(&consumerConfig)

	// Start a goroutine to consume malformed PGCRs
	go malformedWorker(consumerConfig.MalformedChannel, db)

	for {
		if !consumerConfig.GapMode {
			numWorkers = spawnWorkers(numWorkers, db, &consumerConfig)
		} else {
			spawnGapModeWorkers(db, &consumerConfig)
			// When exiting gap mode, we should increase the number of workers to catch up
			numWorkers = maxWorkers
		}
	}

}

func spawnWorkers(countWorkers int, db *sql.DB, consumerConfig *ConsumerConfig) int {
	var wg sync.WaitGroup
	ids := make(chan int64, 5)

	logWorkersStarting(countWorkers, consumerConfig.LatestId)

	// When each worker finishes, it will send its info onto these channels
	resultsChannel := make(chan *WorkerResult, countWorkers)

	for i := 0; i < countWorkers; i++ {
		wg.Add(1)
		go Worker(&wg, ids, resultsChannel, consumerConfig.FailuresChannel, consumerConfig.MalformedChannel, db)
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
		// If we aren't getting 404's, just spike the workers up to ensure we catch up to live ASAP
		newWorkers = int(math.Ceil(float64(countWorkers) * (1 + float64(medianLag-30)/100)))
	} else {
		decreaseFraction := (retryDelayTime / 800 * (fractionNotFound - 0.025)) // do not let workers go below 2.5%
		if decreaseFraction > 0.8 {
			decreaseFraction = 0.8
		}
		// Adjust number of workers for the next period
		newWorkers = int(math.Round(float64(countWorkers) - decreaseFraction*float64(countWorkers)))
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
		go gapModeWorker(&wg, ids, consumerConfig.SuccessChannel, db)
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
