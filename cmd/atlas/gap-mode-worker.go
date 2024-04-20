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

	amqp "github.com/rabbitmq/amqp091-go"
)

func gapModeWorker(wg *sync.WaitGroup, ch chan int64, foundChannel chan int64, db *sql.DB, channel *amqp.Channel) {
	defer wg.Done()
	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	randomVariation := retryDelayTime / 2

	for instanceID := range ch {
		startTime := time.Now()
		notFoundCount := 0
		i := 0
		for {
			reqStartTime := time.Now()
			result, lag := pgcr.FetchAndStorePGCR(client, instanceID, db, channel, proxy, securityKey)
			if result == pgcr.Success {
				endTime := time.Now()
				workerTime := endTime.Sub(startTime).Milliseconds()
				reqTime := endTime.Sub(reqStartTime).Milliseconds()
				log.Printf("[GapMode] Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime, lag.Seconds())
				foundChannel <- instanceID
				break
			} else if result == pgcr.AlreadyExists || result == pgcr.NonRaid {
				foundChannel <- instanceID
				break
			} else if result == pgcr.SystemDisabled {
				time.Sleep(30 * time.Second)
			} else if result == pgcr.InsufficientPrivileges {
				logInsufficentPrivileges(instanceID)
				break
			} else if result == pgcr.InternalError || result == pgcr.BadFormat {
				log.Println("Error parsing data for instanceId", instanceID)
			}

			if notFoundCount > 2 {
				log.Printf("[GapMode] Could not find instance id %d a total of %d times, logging it to the file", instanceID, notFoundCount)
				logMissedInstance(instanceID, startTime, true)
				break
			}
			timeout := time.Duration(2*(retryDelayTime-randomVariation+rand.Intn((2*randomVariation)+1))*notFoundCount) * time.Millisecond
			time.Sleep(timeout)
			i++
		}
	}
}
