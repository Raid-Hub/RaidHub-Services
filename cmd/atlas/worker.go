package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"raidhub/shared/monitoring"
	"raidhub/shared/pgcr"
)

func Worker(wg *sync.WaitGroup, ch chan int64, results chan *WorkerResult, failuresChannel chan int64, malformedChannel chan int64, db *sql.DB) {
	defer wg.Done()

	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	randomVariation := retryDelayTime / 2

	var totalWorkerNotFounds int = 0
	// circular buffer
	var behindHead [circularBufferSize]float64
	cb := 0

	for instanceID := range ch {
		startTime := time.Now()
		notFoundCount := 0
		i := 0
		var result pgcr.PGCRResult
		var lag *time.Duration
		var reqTime time.Duration
		for {
			cb = cb % circularBufferSize
			reqStartTime := time.Now()
			result, lag = pgcr.FetchAndStorePGCR(client, instanceID, db, proxy, securityKey)

			if lag != nil {
				behindHead[cb] = lag.Seconds()
				cb++
			}

			// Handle the result
			if result == pgcr.AlreadyExists || result == pgcr.NonRaid {
				break
			}
			if result == pgcr.Success {
				endTime := time.Now()
				workerTime := endTime.Sub(startTime).Milliseconds()
				reqTime = endTime.Sub(reqStartTime)
				log.Printf("Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime.Milliseconds(), lag.Seconds())
				break
			} else if result == pgcr.NotFound || result == pgcr.InternalError {
				notFoundCount++
			} else if result == pgcr.SystemDisabled {
				time.Sleep(30 * time.Second)
			} else if result == pgcr.InsufficientPrivileges {
				failuresChannel <- instanceID
				logInsufficentPrivileges(instanceID, startTime)
				break
			} else if result == pgcr.BadFormat {
				writeMissedLog(instanceID)
				malformedChannel <- instanceID
				break
			}

			// If we have not found the instance id after some time
			if notFoundCount >= numMisses {
				log.Printf("Could not find instance id %d a total of %d times, logging it to the file", instanceID, notFoundCount)
				failuresChannel <- instanceID
				logMissedInstance(instanceID, startTime, false)
				break
			} else if notFoundCount == numMissesForWarning {
				logMissedInstanceWarning(instanceID, startTime)
			}
			timeout := time.Duration((retryDelayTime - randomVariation + rand.Intn((2*randomVariation)*(i+1)))) * time.Millisecond
			time.Sleep(timeout)
			i++
		}

		statusStr := fmt.Sprintf("%d", result)
		attemptsStr := fmt.Sprintf("%d", i+1)

		monitoring.PGCRCrawlStatus.WithLabelValues(statusStr, attemptsStr).Inc()
		if lag != nil {
			monitoring.PGCRCrawlLag.WithLabelValues(statusStr, attemptsStr).Observe(float64(lag.Seconds()))
		}
		monitoring.PGCRCrawlReqTime.WithLabelValues(statusStr, attemptsStr).Observe(float64(reqTime.Milliseconds()))

		// Track the number of not founds for this worker
		totalWorkerNotFounds += notFoundCount
	}

	results <- &WorkerResult{
		Lag:       behindHead[:],
		NotFounds: totalWorkerNotFounds,
	}
}
