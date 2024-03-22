package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"raidhub/shared/pgcr"
)

func Worker(wg *sync.WaitGroup, ch chan int64, results chan *WorkerResult, failuresChannel chan int64, db *sql.DB) {
	defer wg.Done()

	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	randomVariation := retryDelayTime / 2

	var notFound int = 0
	// circular buffer
	var behindHead [circularBufferSize]float64
	cb := 0

	for instanceID := range ch {
		startTime := time.Now()
		notFoundCount := 0
		i := 0
		for {
			cb = cb % circularBufferSize
			reqStartTime := time.Now()
			result, lag := pgcr.FetchAndStorePGCR(client, instanceID, db, proxy, securityKey)

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
				reqTime := endTime.Sub(reqStartTime).Milliseconds()
				log.Printf("Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime, lag.Seconds())
				break
			} else if result == pgcr.NotFound {
				notFound++
				notFoundCount++
			} else if result == pgcr.SystemDisabled {
				time.Sleep(30 * time.Second)
			} else if result == pgcr.InsufficientPrivileges {
				failuresChannel <- instanceID
				logInsufficentPrivileges(instanceID, startTime)
				break
			} else if result == pgcr.InternalError || result == pgcr.BadFormat {
				log.Println("Error parsing data for instanceId", instanceID)
			}

			// If we have not found the instance id after some time
			if notFoundCount >= numMisses {
				log.Printf("Could not find instance id %d a total of %d times, logging it to the file", instanceID, notFoundCount)
				failuresChannel <- instanceID
				logMissedInstance(instanceID, startTime, false)
				notFound++
				break
			} else if notFoundCount == numMissesForWarning {
				logMissedInstanceWarning(instanceID, startTime)
			}
			timeout := time.Duration((retryDelayTime-randomVariation+rand.Intn((2*randomVariation)+1))*notFoundCount) * time.Millisecond
			time.Sleep(timeout)
			i++
		}
		// Track the number of not founds for this request
		notFoundCount += notFound
	}

	results <- &WorkerResult{
		Lag:       behindHead[:],
		NotFounds: notFound,
	}
}
