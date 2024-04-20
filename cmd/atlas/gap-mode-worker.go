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
			result, activity, raw, err := pgcr.FetchAndProcessPGCR(client, instanceID, proxy, securityKey)
			if err != nil {
				log.Println(err)
			}
			if result == pgcr.NotFound {
				notFoundCount++
			} else if result == pgcr.Success {
				lag, committed, err := pgcr.StorePGCR(activity, raw, db, channel)
				if err != nil {
					log.Println("Error storing data for instanceId", instanceID)
					time.Sleep(5 * time.Second)
				} else {
					if !committed {
						log.Printf("[GapMode] Found duplicate raid with instanceId %d", instanceID)
					} else {
						endTime := time.Now()
						workerTime := endTime.Sub(startTime).Milliseconds()
						reqTime := endTime.Sub(reqStartTime).Milliseconds()
						log.Printf("[GapMode] Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime, lag.Seconds())
					}
					foundChannel <- instanceID
					break
				}
			} else if result == pgcr.NonRaid {
				foundChannel <- instanceID
				break
			} else if result == pgcr.SystemDisabled {
				time.Sleep(30 * time.Second)
				continue
			} else if result == pgcr.InsufficientPrivileges {
				logInsufficentPrivileges(instanceID)
				pgcr.WriteMissedLog(instanceID)
				break
			} else {
				log.Printf("[GapMode] Error type %d for instanceId %d: %s", result, instanceID, err)
			}

			if notFoundCount > 2 || i > 5 {
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
